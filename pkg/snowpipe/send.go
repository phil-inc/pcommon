package snowpipe

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	mrand "math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	attemptTimeout = 30 * time.Second
	maxBackoff     = 5 * time.Second
	retryFor       = time.Minute // how long one request is retried
)

// send posts chunks as one request, halving it while it compresses past the cap. Every chunk
// gets an outcome, even if sending panics.
func (c *Client) send(pipe string, chunks []*chunk) {
	defer func() {
		if p := recover(); p != nil {
			finishAll(chunks, fmt.Errorf("snowpipe: internal error: %v", p))
		}
	}()
	if c.ctx.Err() != nil {
		finishAll(chunks, ErrClosed)
		return
	}
	var raw []byte
	for _, ch := range chunks {
		raw = append(raw, ch.data...)
	}
	body := c.zstd.EncodeAll(raw, nil)
	if len(body) > maxPayload && len(chunks) > 1 {
		half := len(chunks) / 2
		c.send(pipe, chunks[:half])
		c.send(pipe, chunks[half:])
		return
	}
	finishAll(chunks, c.post(&request{
		path: fmt.Sprintf("/v2/streaming/data/databases/%s/schemas/%s/pipes/%s/channels/ELASTIC/rows",
			url.PathEscape(c.cfg.Database), url.PathEscape(c.cfg.Schema), url.PathEscape(pipe)),
		id:    requestID(),
		body:  body,
		raw:   len(raw),
		calls: callers(chunks),
	}))
}

// request is one rowset, sent until it is stored or cannot be.
type request struct {
	path  string
	id    string
	body  []byte  // zstd
	raw   int     // uncompressed size
	calls []*call // callers waiting on its rows
}

// callers lists the calls behind chunks. A call's chunks are queued together, so comparing
// with the previous one is enough.
func callers(chunks []*chunk) []*call {
	var out []*call
	for _, ch := range chunks {
		if len(out) == 0 || out[len(out)-1] != ch.call {
			out = append(out, ch.call)
		}
	}
	return out
}

// post sends r until it is stored or cannot be. Every try has the same requestId and the next
// retryCount, which is how the Elastic API tells a retry from new rows.
func (c *Client) post(r *request) error {
	deadline := time.Now().Add(retryFor)
	backoff := 100 * time.Millisecond
	stale, renewals := "", 0
	for try := 0; ; try++ {
		token, wait, err := c.attempt(r, try, stale)
		if err == nil {
			return nil
		}
		var se *StatusError
		isStatus := errors.As(err, &se)
		switch {
		case c.ctx.Err() != nil:
			return fmt.Errorf("%w: %w", ErrClosed, err)
		case isStatus && se.Status == http.StatusUnauthorized && renewals < 2:
			stale, renewals = token, renewals+1
			continue
		case isStatus && !retryable(se.Status):
			return err
		case abandoned(r.calls):
			return err
		}
		wait = max(wait, backoff/2+mrand.N(backoff/2+1))
		if time.Now().Add(wait).After(deadline) {
			return fmt.Errorf("gave up after %d tries: %w", try+1, err)
		}
		select {
		case <-time.After(wait):
		case <-c.ctx.Done():
			return fmt.Errorf("%w: %w", ErrClosed, err)
		}
		backoff = min(2*backoff, maxBackoff)
	}
}

func retryable(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500
}

// attempt sends one try. It returns the token it used and the Retry-After wait. The try is cut
// off once every caller has returned, so a stalled request frees its worker.
func (c *Client) attempt(r *request, try int, stale string) (string, time.Duration, error) {
	ctx, cancel := context.WithTimeout(c.ctx, attemptTimeout)
	defer cancel()
	var waiting atomic.Int32
	waiting.Store(int32(len(r.calls)))
	for _, cl := range r.calls {
		stop := context.AfterFunc(cl.ctx, func() {
			if waiting.Add(-1) == 0 {
				cancel()
			}
		})
		defer stop()
	}

	base, token, err := c.auth.ingestToken(ctx, stale)
	if err != nil {
		return "", 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s%s?requestId=%s&retryCount=%d", base, r.path, r.id, try), bytes.NewReader(r.body))
	if err != nil {
		return token, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header.Set("Content-Encoding", "zstd")
	// Without it Snowflake answers 400 "Unsupported Accept header null".
	req.Header.Set("Accept", "application/json")
	// Undocumented; the Rust SDK sends it so the server can size the rows up front.
	req.Header.Set("x-snowflake-uncompressed-content-length", strconv.Itoa(r.raw))
	resp, err := c.http.Do(req)
	if err != nil {
		return token, 0, fmt.Errorf("post rows: %w", err)
	}
	wait := retryAfter(resp.Header.Get("Retry-After"))
	_, err = readBody(resp)
	var se *StatusError
	if errors.As(err, &se) {
		se.Message = "" // can quote a row
	}
	return token, wait, err
}

// retryAfter reads Retry-After in seconds.
func retryAfter(v string) time.Duration {
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

// requestID is a random UUID (version 4).
func requestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
