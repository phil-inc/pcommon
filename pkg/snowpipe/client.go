// Package snowpipe is a client for the Snowpipe Streaming REST API and the Snowflake SQL API.
// Every request is HTTPS with a key pair JWT the client signs itself, so there is no session
// that can expire and leave the caller stuck.
//
// AppendRows and AppendTables write rows. They return once Snowflake has stored every row,
// and the client does the rest: request sizing, compression, a fixed pool of NumWorkers
// goroutines, retries and token renewal. See append.go.
package snowpipe

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Config is one Snowflake account, user and target schema.
type Config struct {
	Account    string // the host is <Account>.snowflakecomputing.com
	User       string
	PrivateKey string // PEM, PKCS#8 or PKCS#1, not encrypted
	Database   string
	Schema     string
	Warehouse  string // used by Query only
	Role       string // used by Query only

	// BaseURL replaces https://<Account>.snowflakecomputing.com, for a private link or a test server.
	BaseURL string
	// HTTPClient sends every request. The default has a 1 minute timeout.
	HTTPClient *http.Client

	// NumWorkers is the size of the worker pool that sends rows for every pipe together, and so
	// the most requests in flight: a Go HTTP request holds its goroutine until the response.
	// The pool is fixed for the life of the client: more load makes requests bigger and queues
	// longer, never more goroutines. Default 1.
	NumWorkers int
	// RetryFor bounds how long one request is retried. A request also stops once every
	// AppendRows waiting on it has returned. Default 1 minute.
	RetryFor time.Duration
}

// jwtLife is under the 1 hour Snowflake allows. scopedLife is how long a scoped token is reused;
// a 401 before then gets a new one.
const (
	jwtLife    = 59 * time.Minute
	scopedLife = 30 * time.Minute
)

// Client talks to one Snowflake account. It is safe for concurrent use.
type Client struct {
	cfg         Config
	key         *rsa.PrivateKey
	fingerprint string
	base        string
	http        *http.Client
	ownHTTP     bool // http was built here, so Close may close its idle connections

	mu       sync.Mutex // guards the ingest host and the scoped token
	ingest   string     // https://<ingest host>
	scoped   string
	scopedAt time.Time

	sender // the AppendRows machinery, in append.go
}

// New parses the private key; its error is about the key. It sends no request; call Connect
// to check the account.
func New(cfg Config) (*Client, error) {
	key, err := parseKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	if cfg.Account == "" || cfg.User == "" {
		return nil, errors.New("snowpipe: Account and User are required")
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(der)
	c := &Client{
		cfg:         cfg,
		key:         key,
		fingerprint: "SHA256:" + base64.StdEncoding.EncodeToString(sum[:]),
		base:        strings.TrimSuffix(cfg.BaseURL, "/"),
		http:        cfg.HTTPClient,
	}
	if c.base == "" {
		c.base = "https://" + cfg.Account + ".snowflakecomputing.com"
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: time.Minute, Transport: newTransport()}
		c.ownHTTP = true
	}
	if err := c.sender.init(&c.cfg); err != nil {
		return nil, err
	}
	return c, nil
}

// newTransport is http.DefaultTransport with a larger idle pool per host. Every request goes
// to one of two hosts, and with the default of 2 idle connections per host, runs at 5k rows/s
// stalled for seconds and timed out large uploads. That did not happen with 64.
func newTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 256
	t.MaxIdleConnsPerHost = 64
	return t
}

func parseKey(pemText string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil, errors.New("not a PEM block")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("not an unencrypted PKCS#8 or PKCS#1 key")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA key")
	}
	return key, nil
}

// jwt signs the token of Snowflake's key pair authentication. The account is the part before
// the first dot, so a locator with a region still works.
func (c *Client) jwt() (string, error) {
	account, _, _ := strings.Cut(strings.ToUpper(c.cfg.Account), ".")
	user := account + "." + strings.ToUpper(c.cfg.User)
	now := time.Now()
	claims, err := json.Marshal(map[string]any{
		"iss": user + "." + c.fingerprint,
		"sub": user,
		"iat": now.Unix(),
		"exp": now.Add(jwtLife).Unix(),
	})
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding
	signed := enc.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." + enc.EncodeToString(claims)
	sum := sha256.Sum256([]byte(signed))
	sig, err := rsa.SignPKCS1v15(nil, c.key, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed + "." + enc.EncodeToString(sig), nil
}

// control sends a request to the account host with a new JWT.
func (c *Client) control(ctx context.Context, method, path, contentType string, body io.Reader) (*http.Response, error) {
	token, err := c.jwt()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Snowflake-Authorization-Token-Type", "KEYPAIR_JWT")
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.http.Do(req)
}

// Ping asks the account for its ingest host. It proves the key is accepted and runs no query,
// so the warehouse stays suspended.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.hostname(ctx)
	return err
}

// Connect gets the ingest host and a scoped token, so a bad role or network fails early.
func (c *Client) Connect(ctx context.Context) error {
	_, _, err := c.ingestToken(ctx, "")
	return err
}

func (c *Client) hostname(ctx context.Context) (string, error) {
	resp, err := c.control(ctx, http.MethodGet, "/v2/streaming/hostname", "", nil)
	if err != nil {
		return "", fmt.Errorf("get ingest host: %w", err)
	}
	body, err := readBody(resp)
	if err != nil {
		return "", fmt.Errorf("get ingest host: %w", err)
	}
	// the ingest host needs dashes where the account name has underscores
	return strings.ReplaceAll(strings.TrimSpace(string(body)), "_", "-"), nil
}

// ingestToken returns the ingest base URL and a scoped token for it. stale is a token the ingest
// host answered 401 to: it is replaced, unless another request already replaced it.
func (c *Client) ingestToken(ctx context.Context, stale string) (base, token string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.scoped != "" && c.scoped != stale && time.Since(c.scopedAt) < scopedLife {
		return c.ingest, c.scoped, nil
	}
	if c.ingest == "" {
		host, err := c.hostname(ctx)
		if err != nil {
			return "", "", err
		}
		c.ingest = "https://" + host
	}

	host := strings.TrimPrefix(c.ingest, "https://")
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"scope":      {host},
	}
	resp, err := c.control(ctx, http.MethodPost, "/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("get scoped token: %w", err)
	}
	body, err := readBody(resp)
	if err != nil {
		return "", "", fmt.Errorf("get scoped token: %w", err)
	}
	// the reference shows {"token": ...} and the tutorial a bare token; accept both
	scoped := strings.TrimSpace(string(body))
	if strings.HasPrefix(scoped, "{") {
		var v struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(body, &v); err != nil {
			return "", "", fmt.Errorf("get scoped token: %w", err)
		}
		scoped = v.Token
	}
	if scoped == "" {
		return "", "", errors.New("get scoped token: empty token")
	}
	c.scoped, c.scopedAt = scoped, time.Now()
	return c.ingest, c.scoped, nil
}

// StatusError is a response other than 2xx. Code is Snowflake's error code. Message is
// Snowflake's text. It is empty for appends, because there it can quote a row, and errors
// get logged.
type StatusError struct {
	Status  int
	Code    string
	Message string
}

func (e *StatusError) Error() string {
	s := fmt.Sprintf("snowflake: HTTP %d", e.Status)
	if e.Code != "" {
		s += " " + e.Code
	}
	if e.Message != "" {
		s += ": " + e.Message
	}
	return s
}

// Is makes a 413 match ErrTooLarge.
func (e *StatusError) Is(target error) bool {
	return target == ErrTooLarge && e.Status == http.StatusRequestEntityTooLarge
}

// readBody reads and closes the body of a 2xx response, or returns a *StatusError.
func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode/100 == 2 {
		return body, nil
	}
	var v struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &v) // an error body that is not JSON keeps only the status
	return nil, &StatusError{Status: resp.StatusCode, Code: v.Code, Message: v.Message}
}
