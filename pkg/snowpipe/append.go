package snowpipe

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	mrand "math/rand/v2"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"
)

// maxPayload is Snowflake's cap on one request as sent, after compression.
const maxPayload = 4 << 20

// MaxRow bounds one encoded row. A row under it fits one request even if it does not compress,
// so an oversized row fails before anything is sent, where the caller knows which row it is.
const MaxRow = maxPayload - 64<<10

// maxRequest bounds the uncompressed rows of one request, whatever the compression ratio.
const maxRequest = 16 << 20

// maxQueued bounds the encoded rows held for every pipe together, sent or waiting. A call
// that would go over it waits for room, so a slow or failing Snowflake slows the callers
// down instead of growing memory.
const maxQueued = 64 << 20

var (
	// ErrTooLarge is a row over MaxRow, or a request Snowflake answered 413. Sending it again
	// fails again.
	ErrTooLarge = errors.New("snowpipe: too large")
	// ErrClosed is AppendRows after Close, or rows Close gave up on.
	ErrClosed = errors.New("snowpipe: client closed")
)

// TableError is the rows of one pipe that a call did not store, and why. AppendRows and
// AppendTables return one per failed pipe, joined with errors.Join; TableErrors lists them.
// It never holds row values.
type TableError struct {
	Pipe string
	Rows int // rows of this pipe in the call that may not be stored
	Err  error
}

func (e *TableError) Error() string {
	return fmt.Sprintf("%s: %d rows not stored: %v", e.Pipe, e.Rows, e.Err)
}

func (e *TableError) Unwrap() error { return e.Err }

// TableErrors returns every *TableError in err, which may be a joined or wrapped error.
func TableErrors(err error) []*TableError {
	var out []*TableError
	var walk func(error)
	walk = func(err error) {
		if te, ok := err.(*TableError); ok {
			out = append(out, te)
			return
		}
		switch u := err.(type) {
		case interface{ Unwrap() []error }:
			for _, e := range u.Unwrap() {
				walk(e)
			}
		case interface{ Unwrap() error }:
			if e := u.Unwrap(); e != nil {
				walk(e)
			}
		}
	}
	if err != nil {
		walk(err)
	}
	return out
}

// AppendRows stores the rows in the pipe's Elastic channel and returns once Snowflake has
// acknowledged every one of them. The pipe is in Config.Database and Config.Schema.
//
// It is safe for concurrent use, and concurrent calls to one pipe share requests. Throttling,
// outages, timeouts and expired tokens are retried inside, until ctx ends or RetryFor passes.
// An error means some rows may not be stored; calling again with the same rows can store the
// others twice, so readers must tolerate duplicates. ErrTooLarge comes before anything is sent.
func (c *Client) AppendRows(ctx context.Context, pipe string, rows []map[string]any) error {
	return c.AppendTables(ctx, map[string][]map[string]any{pipe: rows})
}

// AppendTables is AppendRows for several pipes at once. It queues every pipe's rows and then
// waits for all of them, so a caller that writes many tables needs no goroutines of its own.
// Every row is encoded and checked before any is sent.
func (c *Client) AppendTables(ctx context.Context, tables map[string][]map[string]any) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("snowpipe: internal error: %v", p)
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	type encoded struct {
		pipe string
		data []byte
		rows int
	}
	var all []encoded
	release := func() {
		for _, e := range all {
			dataPool.put(e.data)
		}
	}
	for pipe, rows := range tables {
		if len(rows) == 0 {
			continue
		}
		data, err := encode(rows)
		if err != nil {
			release()
			return &TableError{Pipe: strings.ToUpper(pipe), Rows: len(rows), Err: err}
		}
		all = append(all, encoded{pipe, data, len(rows)})
	}
	if len(all) == 0 {
		return nil
	}
	size := 0
	for _, e := range all {
		size += len(e.data)
	}
	if err := c.room.acquire(ctx, size); err != nil {
		release()
		return err
	}

	queues, err := c.begin(len(all), func(i int) string { return all[i].pipe })
	if err != nil {
		c.room.release(size)
		release()
		return err
	}
	defer c.calls.Done()

	w := &waiter{ctx: ctx, done: make(chan struct{}), size: size, room: &c.room}
	cuts := make([][]*chunk, len(all))
	n := 0
	for i, e := range all {
		w.bufs = append(w.bufs, e.data)
		cuts[i] = w.cut(queues[i].name, e.data, e.rows, c.target())
		n += len(cuts[i])
	}
	w.pending.Store(int32(n))
	for i, q := range queues {
		c.enqueue(q, cuts[i])
	}

	select {
	case <-w.done:
		return w.result()
	case <-ctx.Done():
		return ctx.Err() // chunks not yet sent are dropped; one in flight may still land
	}
}

// encode writes the rows as NDJSON in the caller's goroutine, so encoding scales with callers
// and costs no worker time.
func encode(rows []map[string]any) (_ []byte, err error) {
	buf := bytes.NewBuffer(dataPool.get(len(rows) * 256))
	row := 0
	defer func() {
		// a value's MarshalJSON can panic; the panic value is not shown, it can quote the row
		if p := recover(); p != nil {
			dataPool.put(buf.Bytes())
			err = fmt.Errorf("encode row %d: panic in a value's JSON encoding", row)
		}
	}()
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false) // <, > and & are valid JSON; escaping them only costs bytes
	for i, r := range rows {
		row = i
		if r == nil {
			dataPool.put(buf.Bytes())
			return nil, fmt.Errorf("row %d is nil", i)
		}
		before := buf.Len()
		if err := enc.Encode(r); err != nil {
			dataPool.put(buf.Bytes())
			return nil, fmt.Errorf("encode row %d: %w", i, err)
		}
		if n := buf.Len() - before; n > MaxRow {
			dataPool.put(buf.Bytes())
			return nil, fmt.Errorf("row %d is %d bytes, over the request limit: %w", i, n, ErrTooLarge)
		}
	}
	return buf.Bytes(), nil
}

// Close waits for AppendRows calls in progress, then stops the workers. New calls get
// ErrClosed. If ctx ends first, requests in flight are cut off and their callers get an error.
func (c *Client) Close(ctx context.Context) error {
	c.qmu.Lock()
	if c.closed {
		c.qmu.Unlock()
		return nil
	}
	c.closed = true
	c.qmu.Unlock()

	done := make(chan struct{})
	go func() {
		c.calls.Wait()
		c.qmu.Lock()
		c.stopping = true
		c.cond.Broadcast()
		c.qmu.Unlock()
		c.workers.Wait()
		close(done)
	}()
	var err error
	select {
	case <-done:
	case <-ctx.Done():
		err = ctx.Err()
		c.cancel()
		<-done
	}
	c.cancel()
	if c.ownHTTP {
		c.http.CloseIdleConnections()
	}
	return err
}

// Stats are counters since New.
type Stats struct {
	Rows              int64 // rows Snowflake stored
	Requests          int64 // requests Snowflake stored
	Retries           int64 // tries after the first, including token renewals
	Throttled         int64 // 429 responses
	FailedRows        int64 // rows that failed, or were dropped because their caller gave up
	UncompressedBytes int64 // NDJSON bytes of stored requests
	CompressedBytes   int64 // bytes sent for stored requests
}

// Stats returns the counters. It is safe to call at any time, including after Close.
func (c *Client) Stats() Stats {
	return Stats{
		Rows: c.nRows.Load(), Requests: c.nRequests.Load(), Retries: c.nRetries.Load(),
		Throttled: c.nThrottled.Load(), FailedRows: c.nFailed.Load(),
		UncompressedBytes: c.nRaw.Load(), CompressedBytes: c.nWire.Load(),
	}
}

// sender is the part of Client behind AppendRows. Rows wait in one queue per pipe. A fixed
// set of workers takes a pipe off the ready list, packs what that pipe has queued into one
// request near the 4 MB cap, sends it and retries it. A busy pipe gets bigger requests rather
// than more goroutines; an idle one is sent at once.
type sender struct {
	cfg    *Config
	zstd   *zstd.Encoder // EncodeAll is safe for concurrent use
	ctx    context.Context
	cancel context.CancelFunc

	qmu      sync.Mutex
	cond     *sync.Cond // signalled when ready gains a pipe, or on stop
	pipes    map[string]*queue
	ready    []*queue // pipes with queued rows and a free in-flight slot, oldest first
	closed   bool     // no new AppendRows
	stopping bool     // workers exit once ready is empty
	start    sync.Once
	calls    sync.WaitGroup // AppendRows in progress
	workers  sync.WaitGroup
	live     atomic.Int32 // running workers, for tests

	ratio atomic.Uint64 // smoothed uncompressed/compressed, float64 bits
	room  room          // bytes held by calls in progress, capped at maxQueued

	nRows, nRequests, nRetries, nThrottled, nFailed, nRaw, nWire atomic.Int64
}

// queue is one pipe's rows waiting to be sent. Fields are guarded by sender.qmu.
type queue struct {
	name     string
	chunks   []*chunk
	inflight int
	listed   bool // in sender.ready
}

// chunk is whole NDJSON rows of one pipe from one AppendRows.
type chunk struct {
	pipe string
	data []byte
	rows int
	w    *waiter
}

// waiter is one AppendRows until each of its chunks is stored or has failed.
type waiter struct {
	ctx     context.Context
	bufs    [][]byte // the encoded rows, one per pipe; chunks slice into them
	size    int      // bytes of bufs, held in room until the last chunk finishes
	room    *room
	pending atomic.Int32
	mu      sync.Mutex
	failed  map[string]*TableError // by pipe
	done    chan struct{}
}

// result is nil, or one *TableError per pipe that failed, in pipe order.
func (w *waiter) result() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.failed) == 0 {
		return nil
	}
	pipes := make([]string, 0, len(w.failed))
	for p := range w.failed {
		pipes = append(pipes, p)
	}
	sort.Strings(pipes)
	errs := make([]error, len(pipes))
	for i, p := range pipes {
		errs[i] = w.failed[p]
	}
	return errors.Join(errs...)
}

func (s *sender) init(cfg *Config) error {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 1
	}
	if cfg.RetryFor <= 0 {
		cfg.RetryFor = time.Minute
	}
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return err
	}
	s.cfg, s.zstd = cfg, enc
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.cond = sync.NewCond(&s.qmu)
	s.pipes = map[string]*queue{}
	s.room = room{limit: maxQueued, freed: make(chan struct{})}
	s.ratio.Store(math.Float64bits(3)) // a usual ratio for JSON until one is measured
	return nil
}

// begin registers one call on n pipes, named by name(i), and starts the workers on first use.
func (c *Client) begin(n int, name func(int) string) ([]*queue, error) {
	c.qmu.Lock()
	defer c.qmu.Unlock()
	if c.closed {
		return nil, ErrClosed
	}
	c.start.Do(func() {
		for range c.cfg.NumWorkers {
			c.workers.Add(1)
			go c.work()
		}
	})
	out := make([]*queue, n)
	for i := range out {
		key := strings.ToUpper(name(i)) // pipe names are case-insensitive in the REST path
		q := c.pipes[key]
		if q == nil {
			q = &queue{name: key}
			c.pipes[key] = q
		}
		out[i] = q
	}
	c.calls.Add(1)
	return out, nil
}

func (c *Client) enqueue(q *queue, chunks []*chunk) {
	c.qmu.Lock()
	q.chunks = append(q.chunks, chunks...)
	c.list(q)
	c.qmu.Unlock()
}

// perPipe is how many workers one pipe may use at once: half the pool, at least one, so a
// pipe with a backlog drains on several workers while the others still get the rest.
func (s *sender) perPipe() int { return max(1, s.cfg.NumWorkers/2) }

// list puts q on the ready list if it has rows and a free slot. The caller holds qmu.
func (s *sender) list(q *queue) {
	if !q.listed && len(q.chunks) > 0 && q.inflight < s.perPipe() {
		q.listed = true
		s.ready = append(s.ready, q)
		s.cond.Signal()
	}
}

func (c *Client) work() {
	defer c.workers.Done()
	c.live.Add(1)
	defer c.live.Add(-1)
	for {
		c.qmu.Lock()
		for len(c.ready) == 0 && !c.stopping {
			c.cond.Wait()
		}
		if len(c.ready) == 0 {
			c.qmu.Unlock()
			return
		}
		q := c.ready[0]
		c.ready[0] = nil
		c.ready = c.ready[1:]
		q.listed = false
		r := c.take(q, c.target())
		if r != nil {
			q.inflight++
		}
		c.list(q) // more rows left: another worker can take them now
		c.qmu.Unlock()
		if r == nil {
			continue
		}

		if c.ctx.Err() != nil {
			c.fail(r, ErrClosed)
		} else {
			c.send(q.name, r)
		}

		c.qmu.Lock()
		q.inflight--
		c.list(q)
		c.qmu.Unlock()
	}
}

// take removes up to size bytes of q's chunks, always at least one, and drops chunks whose
// caller has given up. The caller holds qmu.
func (s *sender) take(q *queue, size int) *request {
	r := &request{}
	for len(q.chunks) > 0 {
		ch := q.chunks[0]
		if err := ch.w.ctx.Err(); err != nil {
			q.pop()
			s.nFailed.Add(int64(ch.rows))
			ch.finish(err)
			continue
		}
		if len(r.chunks) > 0 && r.size+len(ch.data) > size {
			break
		}
		q.pop()
		r.add(ch)
	}
	if len(r.chunks) == 0 {
		return nil
	}
	return r
}

func (q *queue) pop() {
	q.chunks[0] = nil
	q.chunks = q.chunks[1:]
}

// cut splits one pipe's rows into pieces of at most size bytes on row boundaries; a row over
// size gets a piece of its own.
func (w *waiter) cut(pipe string, data []byte, rows, size int) []*chunk {
	if len(data) <= size {
		return []*chunk{{pipe: pipe, data: data, rows: rows, w: w}}
	}
	var out []*chunk
	for rest := data; len(rest) > 0; {
		n := len(rest)
		if n > size {
			if n = bytes.LastIndexByte(rest[:size], '\n') + 1; n == 0 {
				n = bytes.IndexByte(rest, '\n') + 1
			}
		}
		out = append(out, &chunk{pipe: pipe, data: rest[:n], rows: bytes.Count(rest[:n], newline), w: w})
		rest = rest[n:]
	}
	return out
}

var newline = []byte{'\n'}

// finish records the chunk's outcome on its call; the last chunk frees the rows and wakes
// AppendRows.
func (ch *chunk) finish(err error) {
	w := ch.w
	if err != nil {
		w.mu.Lock()
		if w.failed == nil {
			w.failed = map[string]*TableError{}
		}
		if te := w.failed[ch.pipe]; te != nil {
			te.Rows += ch.rows
		} else {
			w.failed[ch.pipe] = &TableError{Pipe: ch.pipe, Rows: ch.rows, Err: err}
		}
		w.mu.Unlock()
	}
	if w.pending.Add(-1) == 0 {
		for _, b := range w.bufs {
			dataPool.put(b)
		}
		w.bufs = nil
		w.room.release(w.size)
		close(w.done)
	}
}

// request is the chunks that go out in one HTTP request.
type request struct {
	chunks   []*chunk
	size     int
	rows     int
	finished bool // its chunks are finished; only the worker sending it touches this
}

func (r *request) add(ch *chunk) {
	r.chunks = append(r.chunks, ch)
	r.size += len(ch.data)
	r.rows += ch.rows
}

func (r *request) finish(err error) {
	if r.finished {
		return
	}
	r.finished = true
	for _, ch := range r.chunks {
		ch.finish(err)
	}
}

// abandoned reports whether every AppendRows waiting on r has returned.
func (r *request) abandoned() bool {
	for _, ch := range r.chunks {
		if ch.w.ctx.Err() == nil {
			return false
		}
	}
	return true
}

// split halves r between chunks, or inside its one chunk on a row boundary.
func (r *request) split() (*request, *request, bool) {
	left, right := &request{}, &request{}
	if len(r.chunks) > 1 {
		half := len(r.chunks) / 2
		for _, ch := range r.chunks[:half] {
			left.add(ch)
		}
		for _, ch := range r.chunks[half:] {
			right.add(ch)
		}
		r.finished = true // its chunks now live in left and right, which finish them
		return left, right, true
	}
	ch := r.chunks[0]
	mid := len(ch.data) / 2
	cut := bytes.IndexByte(ch.data[mid:], '\n') + mid + 1
	if cut <= mid || cut >= len(ch.data) {
		if cut = bytes.LastIndexByte(ch.data[:mid], '\n') + 1; cut == 0 {
			return nil, nil, false
		}
	}
	ch.w.pending.Add(1) // one chunk becomes two
	a := &chunk{pipe: ch.pipe, data: ch.data[:cut], rows: bytes.Count(ch.data[:cut], newline), w: ch.w}
	left.add(a)
	right.add(&chunk{pipe: ch.pipe, data: ch.data[cut:], rows: ch.rows - a.rows, w: ch.w})
	r.finished = true // its chunks now live in left and right, which finish them
	return left, right, true
}

func (s *sender) target() int {
	r := math.Float64frombits(s.ratio.Load())
	return min(int(0.9*maxPayload*r), maxRequest)
}

// observe folds one request into the smoothed ratio; a lost update under contention only
// leaves the estimate a little stale.
func (s *sender) observe(raw, wire int) {
	if wire <= 0 {
		return
	}
	r := max(float64(raw)/float64(wire), 1)
	old := math.Float64frombits(s.ratio.Load())
	s.ratio.Store(math.Float64bits(0.8*old + 0.2*r))
}

// send compresses r, splits it if it is still over the cap, and posts it. A panic fails r's
// rows with an error instead of stopping the worker or the process.
func (c *Client) send(pipe string, r *request) {
	defer func() {
		if p := recover(); p != nil {
			c.fail(r, fmt.Errorf("snowpipe: internal error: %v", p))
		}
	}()
	data := r.chunks[0].data
	var joined []byte
	if len(r.chunks) > 1 {
		joined = dataPool.get(r.size)
		for _, ch := range r.chunks {
			joined = append(joined, ch.data...)
		}
		data = joined
	}
	body := c.zstd.EncodeAll(data, compPool.get(len(data)/4))
	c.observe(len(data), len(body))
	if joined != nil {
		dataPool.put(joined)
	}
	if len(body) > maxPayload {
		compPool.put(body)
		left, right, ok := r.split()
		if !ok {
			c.fail(r, ErrTooLarge) // cannot happen while MaxRow is under maxPayload
			return
		}
		c.send(pipe, left)
		c.send(pipe, right)
		return
	}
	if err := c.post(pipe, body, len(data), r); err != nil {
		c.fail(r, err)
		return
	}
	c.nRows.Add(int64(r.rows))
	c.nRequests.Add(1)
	c.nRaw.Add(int64(len(data)))
	c.nWire.Add(int64(len(body)))
	r.finish(nil)
}

func (c *Client) fail(r *request, err error) {
	if r.finished {
		return
	}
	c.nFailed.Add(int64(r.rows))
	r.finish(err)
}

// post sends one request. Every attempt has the same requestId and the next retryCount, as
// the Elastic API asks, so Snowflake can tell a retry from new rows.
func (c *Client) post(pipe string, body []byte, raw int, r *request) error {
	path := fmt.Sprintf("/v2/streaming/data/databases/%s/schemas/%s/pipes/%s/channels/ELASTIC/rows",
		url.PathEscape(c.cfg.Database), url.PathEscape(c.cfg.Schema), url.PathEscape(pipe))
	id := requestID()
	deadline := time.Now().Add(c.cfg.RetryFor)
	backoff := 100 * time.Millisecond
	stale, renewals := "", 0
	for try := 0; ; try++ {
		token, wait, err := c.attempt(r, path, id, try, stale, body, raw)
		if err == nil {
			if try == 0 { // after a failed try net/http may still be reading body
				compPool.put(body)
			}
			return nil
		}
		var se *StatusError
		isStatus := errors.As(err, &se)
		switch {
		case c.ctx.Err() != nil:
			return fmt.Errorf("%w: %w", ErrClosed, err)
		case isStatus && se.Status == http.StatusUnauthorized && renewals < 2:
			stale, renewals = token, renewals+1
			c.nRetries.Add(1)
			continue
		case isStatus && !retryable(se.Status):
			return err
		case r.abandoned():
			return err
		}
		if isStatus && se.Status == http.StatusTooManyRequests {
			c.nThrottled.Add(1)
		}
		wait = max(wait, backoff/2+mrand.N(backoff/2+1))
		if time.Now().Add(wait).After(deadline) {
			return fmt.Errorf("gave up after %d tries: %w", try+1, err)
		}
		c.nRetries.Add(1)
		t := time.NewTimer(wait)
		select {
		case <-t.C:
		case <-c.ctx.Done():
			t.Stop()
			return fmt.Errorf("%w: %w", ErrClosed, err)
		}
		backoff = min(backoff*2, 5*time.Second)
	}
}

// retryable is the statuses that can succeed if sent again; other 4xx fail the same way.
func retryable(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500
}

// attempt sends one try and returns the token it used and how long Snowflake asked to wait.
// The try is cut off when every AppendRows waiting on r has returned, so a stalled request
// does not hold a worker for its whole timeout.
func (c *Client) attempt(r *request, path, id string, try int, stale string, body []byte, raw int) (string, time.Duration, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	for _, ch := range r.chunks {
		stop := context.AfterFunc(ch.w.ctx, func() {
			if r.abandoned() {
				cancel()
			}
		})
		defer stop()
	}
	base, token, err := c.ingestToken(ctx, stale)
	if err != nil {
		return "", 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s%s?requestId=%s&retryCount=%d", base, path, id, try), bytes.NewReader(body))
	if err != nil {
		return token, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header.Set("Content-Encoding", "zstd")
	// Snowflake answers 400 "Unsupported Accept header null" without it.
	req.Header.Set("Accept", "application/json")
	// Not in the REST docs; the Rust SDK sends it so the server can size the rows up front.
	req.Header.Set("x-snowflake-uncompressed-content-length", strconv.Itoa(raw))
	resp, err := c.http.Do(req)
	if err != nil {
		return token, 0, fmt.Errorf("post rows: %w", err)
	}
	wait := retryAfter(resp.Header.Get("Retry-After"))
	_, err = readBody(resp)
	var se *StatusError
	if errors.As(err, &se) {
		se.Message = "" // can quote a row; the status and code are enough to act on
	}
	return token, wait, err
}

// retryAfter reads Retry-After as seconds or an HTTP date.
func retryAfter(v string) time.Duration {
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(time.Until(t), 0)
	}
	return 0
}

// requestID is a random UUID (version 4).
func requestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read never fails
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// room is a byte budget that callers wait on. acquire blocks until n bytes fit, or ctx ends.
// A call larger than the whole budget still runs when nothing else holds any, so it cannot
// wait forever.
type room struct {
	mu    sync.Mutex
	used  int
	limit int
	freed chan struct{} // closed and replaced on every release, to wake waiters
}

func (r *room) acquire(ctx context.Context, n int) error {
	for {
		r.mu.Lock()
		if r.used == 0 || r.used+n <= r.limit {
			r.used += n
			r.mu.Unlock()
			return nil
		}
		freed := r.freed
		r.mu.Unlock()
		select {
		case <-freed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *room) release(n int) {
	r.mu.Lock()
	r.used -= n
	close(r.freed)
	r.freed = make(chan struct{})
	r.mu.Unlock()
}

func (r *room) held() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.used
}

// bufPool recycles byte slices; buffers over 32 MB are left to the GC.
type bufPool struct{ p sync.Pool }

var dataPool, compPool bufPool

func (bp *bufPool) get(n int) []byte {
	if v, ok := bp.p.Get().(*[]byte); ok && cap(*v) >= n {
		return (*v)[:0]
	}
	return make([]byte, 0, n)
}

func (bp *bufPool) put(b []byte) {
	if cap(b) > 0 && cap(b) <= 32<<20 {
		bp.p.Put(&b)
	}
}
