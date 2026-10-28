package snowpipe

import (
	"context"
	"sync"
)

// pipeQueue is one pipe's chunks waiting for a worker. Guarded by Client.mu.
type pipeQueue struct {
	name     string
	chunks   []*chunk
	inflight int
	ready    bool // in Client.ready
}

// enqueue cuts the call's rows into chunks and queues them. Workers see none of them until
// c.mu is released, so pending is set in time.
func (c *Client) enqueue(cl *call, pipes []encodedPipe) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	c.calls.Add(1)

	n := 0
	for _, p := range pipes {
		q := c.pipes[p.pipe]
		if q == nil {
			q = &pipeQueue{name: p.pipe}
			c.pipes[p.pipe] = q
		}
		chunks := cut(cl, p)
		n += len(chunks)
		q.chunks = append(q.chunks, chunks...)
		c.markReady(q)
	}
	cl.pending.Store(int32(n))
	return nil
}

// perPipe is how many workers one pipe may use, so a backlog on one pipe cannot starve the rest.
func (c *Client) perPipe() int { return max(1, c.cfg.NumWorkers/2) }

func (c *Client) markReady(q *pipeQueue) {
	if !q.ready && len(q.chunks) > 0 && q.inflight < c.perPipe() {
		q.ready = true
		c.ready = append(c.ready, q)
		c.cond.Signal()
	}
}

func (c *Client) work() {
	defer c.workers.Done()
	for {
		q, chunks, ok := c.next()
		if !ok {
			return
		}
		c.send(q.name, chunks)
		c.mu.Lock()
		q.inflight--
		c.markReady(q)
		c.mu.Unlock()
	}
}

// next waits for a ready pipe and takes up to one request's worth of its chunks. It returns
// false once Close has stopped the workers and nothing is left.
func (c *Client) next() (*pipeQueue, []*chunk, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		for len(c.ready) == 0 && !c.stopping {
			c.cond.Wait()
		}
		if len(c.ready) == 0 {
			return nil, nil, false
		}
		q := c.ready[0]
		c.ready = c.ready[1:]
		q.ready = false
		chunks := q.take(maxRequest)
		if len(chunks) > 0 {
			q.inflight++
		}
		c.markReady(q)
		if len(chunks) > 0 {
			return q, chunks, true
		}
	}
}

// take removes up to size bytes of chunks, always at least one, dropping chunks whose caller
// has returned.
func (q *pipeQueue) take(size int) []*chunk {
	var out []*chunk
	total := 0
	for len(q.chunks) > 0 {
		ch := q.chunks[0]
		if err := ch.call.ctx.Err(); err != nil {
			q.chunks = q.chunks[1:]
			ch.finish(err)
			continue
		}
		if len(out) > 0 && total+len(ch.data) > size {
			break
		}
		q.chunks = q.chunks[1:]
		out = append(out, ch)
		total += len(ch.data)
	}
	return out
}

// budget caps the bytes held by appends in progress, so a slow Snowflake makes callers wait
// instead of growing memory.
type budget struct {
	mu    sync.Mutex
	used  int
	limit int
	freed chan struct{} // closed and replaced on every release
}

func newBudget(limit int) budget { return budget{limit: limit, freed: make(chan struct{})} }

// acquire waits until n bytes fit. A call bigger than the whole budget runs alone, rather than
// never.
func (b *budget) acquire(ctx context.Context, n int) error {
	for {
		b.mu.Lock()
		if b.used == 0 || b.used+n <= b.limit {
			b.used += n
			b.mu.Unlock()
			return nil
		}
		freed := b.freed
		b.mu.Unlock()
		select {
		case <-freed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (b *budget) release(n int) {
	b.mu.Lock()
	b.used -= n
	close(b.freed)
	b.freed = make(chan struct{})
	b.mu.Unlock()
}
