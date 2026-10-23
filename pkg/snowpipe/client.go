// Package snowpipe writes rows to Snowflake through Snowpipe Streaming (Elastic Channels) and
// runs setup statements through the SQL API. See README.md.
package snowpipe

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

// Config is one Snowflake account, user and target schema.
type Config struct {
	Account    string // the host is <Account>.snowflakecomputing.com
	User       string
	PrivateKey string // PEM, PKCS#8, not encrypted
	Database   string
	Schema     string
	Warehouse  string // Query only
	Role       string // Query only

	BaseURL    string       // replaces https://<Account>.snowflakecomputing.com
	HTTPClient *http.Client // default: 1 minute timeout, 64 idle connections per host

	NumWorkers int // goroutines sending rows, and so requests in flight; default 1
}

// Client is safe for concurrent use. Create one per schema and share it.
type Client struct {
	cfg    Config
	http   *http.Client
	auth   auth
	zstd   *zstd.Encoder
	budget budget

	// ctx is cancelled by Close to cut off requests in flight.
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex // guards the fields below
	cond     *sync.Cond // signalled when ready gains a pipe, or on stop
	pipes    map[string]*pipeQueue
	ready    []*pipeQueue
	closed   bool
	stopping bool

	calls   sync.WaitGroup
	workers sync.WaitGroup
}

// New checks the config, parses the key and starts the workers; Close stops them. It sends no
// request; Connect does.
func New(cfg Config) (*Client, error) {
	if cfg.Account == "" || cfg.User == "" || cfg.Database == "" || cfg.Schema == "" {
		return nil, errors.New("snowpipe: Account, User, Database and Schema are required")
	}
	key, err := parseKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("snowpipe: private key: %w", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	fp := sha256.Sum256(der)
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return nil, err
	}
	cfg.NumWorkers = max(cfg.NumWorkers, 1)

	c := &Client{cfg: cfg, http: cfg.HTTPClient, zstd: enc, pipes: map[string]*pipeQueue{}}
	if c.http == nil {
		c.http = &http.Client{Timeout: time.Minute, Transport: newTransport()}
	}
	base := strings.TrimSuffix(cfg.BaseURL, "/")
	if base == "" {
		base = "https://" + cfg.Account + ".snowflakecomputing.com"
	}
	c.auth = auth{
		cfg:         &c.cfg,
		http:        c.http,
		base:        base,
		key:         key,
		fingerprint: "SHA256:" + base64.StdEncoding.EncodeToString(fp[:]),
		lock:        make(chan struct{}, 1),
	}
	c.budget = newBudget(maxQueued)
	c.cond = sync.NewCond(&c.mu)
	c.ctx, c.cancel = context.WithCancel(context.Background())
	for range cfg.NumWorkers {
		c.workers.Add(1)
		go c.work()
	}
	return c, nil
}

// newTransport raises the idle pool per host from 2: with 2, requests at a few thousand rows/s
// stalled for seconds while new connections were set up. A replaced DefaultTransport is used as is.
func newTransport() http.RoundTripper {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return http.DefaultTransport
	}
	t = t.Clone()
	t.MaxIdleConns = 256
	t.MaxIdleConnsPerHost = 64
	return t
}

// Ping proves the key is accepted without running a query, so the warehouse stays suspended.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.auth.hostname(ctx)
	return err
}

// Connect gets the ingest host and a scoped token, so a bad role or network fails early.
func (c *Client) Connect(ctx context.Context) error {
	_, _, err := c.auth.ingestToken(ctx, "")
	return err
}

// Close waits for appends in progress, then stops the workers; later appends get ErrClosed.
// If ctx ends first, requests in flight are cut off and their callers get ErrClosed.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	stop := context.AfterFunc(ctx, c.cancel)
	c.calls.Wait()
	c.mu.Lock()
	c.stopping = true
	c.cond.Broadcast()
	c.mu.Unlock()
	c.workers.Wait()

	cutOff := !stop() // ctx ended before the workers did
	c.cancel()
	if cutOff {
		return ctx.Err()
	}
	return nil
}
