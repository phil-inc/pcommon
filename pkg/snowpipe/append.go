package snowpipe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	maxPayload = 4 << 20  // Snowflake's cap on one request body, after compression
	maxRequest = 16 << 20 // most uncompressed bytes in one request
	maxQueued  = 64 << 20 // most encoded bytes held by appends in progress
)

// MaxRow is the largest encoded row. Rows under it fit one request uncompressed, so a chunk of
// them never needs splitting.
const MaxRow = maxPayload - 64<<10

// AppendRows stores rows in pipe and returns once Snowflake has acknowledged all of them.
func (c *Client) AppendRows(ctx context.Context, pipe string, rows []map[string]any) error {
	return c.AppendTables(ctx, map[string][]map[string]any{pipe: rows})
}

// AppendTables stores the rows of several pipes and returns once Snowflake has acknowledged
// all of them. Concurrent calls to a pipe share requests; failures are retried until ctx ends
// or RetryFor passes. On error some rows may be stored, so a retry can duplicate them. A row
// that cannot be encoded fails the call before anything is sent.
func (c *Client) AppendTables(ctx context.Context, tables map[string][]map[string]any) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("snowpipe: internal error: %v", p)
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	encoded, size, err := encodeTables(tables)
	if err != nil || len(encoded) == 0 {
		return err
	}
	if err := c.budget.acquire(ctx, size); err != nil {
		return err
	}
	call := &call{ctx: ctx, size: size, budget: &c.budget, done: make(chan struct{})}
	if err := c.enqueue(call, encoded); err != nil {
		c.budget.release(size)
		return err
	}
	defer c.calls.Done()

	select {
	case <-call.done:
		return call.result()
	case <-ctx.Done():
		return ctx.Err() // queued chunks are dropped; one already in flight may still land
	}
}

type encodedPipe struct {
	pipe string
	data []byte // NDJSON
}

func encodeTables(tables map[string][]map[string]any) ([]encodedPipe, int, error) {
	var out []encodedPipe
	size := 0
	for pipe, rows := range tables {
		if len(rows) == 0 {
			continue
		}
		pipe = strings.ToUpper(pipe) // pipe names are case-insensitive
		data, err := encode(rows)
		if err != nil {
			return nil, 0, &AppendError{Pipes: []PipeError{{Pipe: pipe, Rows: len(rows), Err: err}}}
		}
		out = append(out, encodedPipe{pipe, data})
		size += len(data)
	}
	return out, size, nil
}

func encode(rows []map[string]any) (_ []byte, err error) {
	var buf bytes.Buffer
	i := 0
	defer func() {
		if p := recover(); p != nil { // from a value's MarshalJSON; p can quote the row
			err = fmt.Errorf("encode row %d: panic in its JSON encoding", i)
		}
	}()
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for ; i < len(rows); i++ {
		if rows[i] == nil {
			return nil, fmt.Errorf("row %d is nil", i)
		}
		before := buf.Len()
		if err := enc.Encode(rows[i]); err != nil {
			return nil, fmt.Errorf("encode row %d: %w", i, err)
		}
		if n := buf.Len() - before; n > MaxRow {
			return nil, fmt.Errorf("row %d is %d bytes: %w", i, n, ErrTooLarge)
		}
	}
	return buf.Bytes(), nil
}

// call is one AppendTables until each of its chunks is stored or has failed.
type call struct {
	ctx     context.Context
	size    int
	budget  *budget
	pending atomic.Int32
	done    chan struct{}

	mu     sync.Mutex
	failed map[string]*PipeError
}

func (c *call) result() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.failed) == 0 {
		return nil
	}
	e := &AppendError{}
	for _, p := range c.failed {
		e.Pipes = append(e.Pipes, *p)
	}
	slices.SortFunc(e.Pipes, func(a, b PipeError) int { return strings.Compare(a.Pipe, b.Pipe) })
	return e
}

// chunk is whole NDJSON rows of one pipe from one call. Once queued it is owned by the queue,
// then by the one worker that takes it, so finished needs no lock.
type chunk struct {
	call     *call
	pipe     string
	data     []byte
	rows     int
	finished bool
}

// cut splits one pipe's rows into chunks of at most MaxRow bytes, on row boundaries.
func cut(cl *call, p encodedPipe) []*chunk {
	var out []*chunk
	for rest := p.data; len(rest) > 0; {
		n := len(rest)
		if n > MaxRow {
			n = bytes.LastIndexByte(rest[:MaxRow], '\n') + 1
		}
		out = append(out, &chunk{call: cl, pipe: p.pipe, data: rest[:n], rows: bytes.Count(rest[:n], []byte{'\n'})})
		rest = rest[n:]
	}
	return out
}

// finish records the chunk's outcome once; the call's last chunk wakes AppendTables.
func (ch *chunk) finish(err error) {
	if ch.finished {
		return
	}
	ch.finished = true
	cl := ch.call
	if err != nil {
		cl.mu.Lock()
		if cl.failed == nil {
			cl.failed = map[string]*PipeError{}
		}
		if p := cl.failed[ch.pipe]; p != nil {
			p.Rows += ch.rows
		} else {
			cl.failed[ch.pipe] = &PipeError{Pipe: ch.pipe, Rows: ch.rows, Err: err}
		}
		cl.mu.Unlock()
	}
	if cl.pending.Add(-1) == 0 {
		cl.budget.release(cl.size)
		close(cl.done)
	}
}

func finishAll(chunks []*chunk, err error) {
	for _, ch := range chunks {
		ch.finish(err)
	}
}

// abandoned reports whether every call waiting on chunks has returned.
func abandoned(chunks []*chunk) bool {
	for _, ch := range chunks {
		if ch.call.ctx.Err() == nil {
			return false
		}
	}
	return true
}
