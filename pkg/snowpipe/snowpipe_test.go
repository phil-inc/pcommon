package snowpipe

import (
	"bufio"
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
)

var testKey = func() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
}()

func testPEM(t *testing.T) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(testKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

type appendReq struct {
	pipe, requestID, retryCount, token string
	rows                               []map[string]any
	wire                               int
}

// fake is the account host and the ingest host of one Snowflake account. appendStatus is the
// status of each append in turn; after the list, appends get 200. delay holds every append.
type fake struct {
	mu           sync.Mutex
	tokens       int
	appends      []appendReq
	appendStatus []int
	stall        bool
	delay        time.Duration
	down         atomic.Bool   // every append gets 503, like an outage
	tokenGate    chan struct{} // if set, token requests wait for it to close

	inflight, peak         atomic.Int64
	pipeInflight, pipePeak sync.Map // pipe -> *atomic.Int64
}

func (f *fake) handler(t *testing.T, host func() string) http.Handler {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/streaming/hostname", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || r.Header.Get("X-Snowflake-Authorization-Token-Type") != "KEYPAIR_JWT" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, host())
	})
	mux.HandleFunc("POST /oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if f.tokenGate != nil {
			select {
			case <-f.tokenGate:
			case <-r.Context().Done():
				return
			}
		}
		if err := r.ParseForm(); err != nil || r.Form.Get("scope") != host() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.tokens++
		n := f.tokens
		f.mu.Unlock()
		fmt.Fprintf(w, "tok%d", n)
	})
	mux.HandleFunc("POST /v2/streaming/data/databases/DB/schemas/SC/pipes/{pipe}/channels/ELASTIC/rows", func(w http.ResponseWriter, r *http.Request) {
		pipe := r.PathValue("pipe")
		defer track(&f.inflight, &f.peak)()
		cur, _ := f.pipeInflight.LoadOrStore(pipe, new(atomic.Int64))
		peak, _ := f.pipePeak.LoadOrStore(pipe, new(atomic.Int64))
		defer track(cur.(*atomic.Int64), peak.(*atomic.Int64))()

		body, _ := io.ReadAll(r.Body)
		if f.stall {
			<-r.Context().Done()
			return
		}
		time.Sleep(f.delay)
		if f.down.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"code": "ERR_TEST", "message": "down"}`)
			return
		}
		if r.Header.Get("Content-Type") != "application/x-ndjson" || r.Header.Get("Content-Encoding") != "zstd" || r.Header.Get("Accept") != "application/json" {
			t.Errorf("headers = %v", r.Header)
		}
		raw, err := dec.DecodeAll(body, nil)
		if err != nil {
			t.Errorf("body is not zstd: %v", err)
			return
		}
		req := appendReq{
			pipe:       pipe,
			requestID:  r.URL.Query().Get("requestId"),
			retryCount: r.URL.Query().Get("retryCount"),
			token:      r.Header.Get("Authorization"),
			wire:       len(body),
		}
		sc := bufio.NewScanner(bytes.NewReader(raw))
		sc.Buffer(nil, maxRow+1)
		for sc.Scan() {
			var row map[string]any
			if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
				t.Errorf("a line is not JSON: %v", err)
			}
			req.rows = append(req.rows, row)
		}
		f.mu.Lock()
		f.appends = append(f.appends, req)
		status := http.StatusOK
		if len(f.appends) <= len(f.appendStatus) {
			status = f.appendStatus[len(f.appends)-1]
		}
		f.mu.Unlock()
		if status == http.StatusOK {
			fmt.Fprint(w, `{"message": "OK"}`)
			return
		}
		w.WriteHeader(status)
		fmt.Fprint(w, `{"code": "ERR_TEST", "message": "row is bad"}`)
	})
	return mux
}

// track counts one request in cur and keeps the peak.
func track(cur, peak *atomic.Int64) func() {
	n := cur.Add(1)
	for m := peak.Load(); n > m && !peak.CompareAndSwap(m, n); m = peak.Load() {
	}
	return func() { cur.Add(-1) }
}

// stored is the rows the fake accepted (status 200), by pipe.
func (f *fake) stored() map[string][]map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string][]map[string]any{}
	for i, a := range f.appends {
		if i < len(f.appendStatus) && f.appendStatus[i] != http.StatusOK {
			continue
		}
		out[a.pipe] = append(out[a.pipe], a.rows...)
	}
	return out
}

func (f *fake) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.appends)
}

func newClient(t *testing.T, f *fake, mod ...func(*Config)) *Client {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewUnstartedServer(f.handler(t, func() string { return srv.Listener.Addr().String() }))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)
	cfg := Config{Account: "myorg-acct", User: "loader", PrivateKey: testPEM(t), Database: "DB", Schema: "SC", BaseURL: srv.URL, HTTPClient: srv.Client()}
	for _, m := range mod {
		m(&cfg)
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close(context.Background()) })
	return c
}

func appendRows(c *Client, ctx context.Context, pipe string, rows []map[string]any) error {
	return c.AppendTables(ctx, map[string][]map[string]any{pipe: rows})
}

func (b *budget) held() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.used
}

func TestAppendSendsEveryRowInOrder(t *testing.T) {
	f := &fake{}
	c := newClient(t, f)
	rows := []map[string]any{
		{"id": "1", "v": "a"},
		{"id": "2", "v": nil},
	}
	if err := appendRows(c, context.Background(), "rx_order", rows); err != nil {
		t.Fatal(err)
	}
	if len(f.appends) != 1 || f.appends[0].pipe != "RX_ORDER" {
		t.Fatalf("appends = %+v, want one request to RX_ORDER", f.appends)
	}
	got := f.appends[0].rows
	if len(got) != 2 || got[0]["v"] != "a" || got[1]["id"] != "2" || got[1]["v"] != nil {
		t.Errorf("rows = %v", got)
	}
}

func TestAppendResponses(t *testing.T) {
	cases := []struct {
		name     string
		status   []int
		ok       bool
		tooLarge bool
		tries    int
	}{
		{"an expired token is renewed and the same rowset sent again", []int{401, 200}, true, false, 2},
		{"a third 401 in a row goes back to the caller", []int{401, 401, 401}, false, false, 3},
		{"a body too large is ErrTooLarge", []int{413}, false, true, 1},
		{"a bad request goes back to the caller", []int{400}, false, false, 1},
		{"a missing pipe goes back to the caller", []int{404}, false, false, 1},
		{"an outage is retried", []int{503, 500}, true, false, 3},
		{"throttling is retried", []int{429}, true, false, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fake{appendStatus: tc.status}
			c := newClient(t, f)
			err := appendRows(c, context.Background(), "T", []map[string]any{{"id": "1"}})
			if (err == nil) != tc.ok || errors.Is(err, ErrTooLarge) != tc.tooLarge || len(f.appends) != tc.tries {
				t.Fatalf("err = %v, tries = %d; want ok %v, too large %v, %d tries", err, len(f.appends), tc.ok, tc.tooLarge, tc.tries)
			}
			var se *StatusError
			if err != nil && (!errors.As(err, &se) || se.Code != "ERR_TEST") {
				t.Errorf("err = %v, want a *StatusError with the Snowflake code", err)
			}
			for i, a := range f.appends {
				if a.requestID != f.appends[0].requestID || a.retryCount != fmt.Sprint(i) {
					t.Errorf("try %d = %+v, want requestId %s and retryCount %d", i, a, f.appends[0].requestID, i)
				}
			}
			if tc.status[0] == http.StatusUnauthorized && f.appends[0].token == f.appends[1].token {
				t.Error("a 401 did not get a new token")
			}
		})
	}
}

func TestScopedTokenIsReusedAcrossAppends(t *testing.T) {
	f := &fake{}
	c := newClient(t, f)
	for range 3 {
		if err := appendRows(c, context.Background(), "T", []map[string]any{{"id": "1"}}); err != nil {
			t.Fatal(err)
		}
	}
	if f.tokens != 1 {
		t.Errorf("tokens = %d, want 1", f.tokens)
	}
}

func TestStalledAppendEndsAtTheDeadline(t *testing.T) {
	f := &fake{stall: true}
	c := newClient(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := appendRows(c, ctx, "T", []map[string]any{{"id": "1"}})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 5*time.Second {
		t.Fatalf("err = %v after %v, want the deadline", err, time.Since(start))
	}
}

// A retried request stops once nobody waits for it, so a caller's timeout bounds the load.
func TestRetriesStopOnceTheCallerHasGivenUp(t *testing.T) {
	f := &fake{appendStatus: make([]int, 1000)}
	for i := range f.appendStatus {
		f.appendStatus[i] = http.StatusServiceUnavailable
	}
	c := newClient(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if err := appendRows(c, ctx, "T", []map[string]any{{"id": "1"}}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	time.Sleep(time.Second) // a backoff already started may still send once
	before := f.count()
	time.Sleep(2 * time.Second)
	if after := f.count(); after != before {
		t.Fatalf("still retrying after the caller left: %d then %d requests", before, after)
	}
}

func TestBadInputFailsBeforeAnythingIsSent(t *testing.T) {
	f := &fake{}
	c := newClient(t, f)
	cases := map[string][]map[string]any{
		"a nil row":                 {{"id": "1"}, nil},
		"a value JSON cannot write": {{"id": "1", "v": make(chan int)}},
	}
	for name, rows := range cases {
		if err := appendRows(c, context.Background(), "T", rows); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	for _, cfg := range []Config{
		{Account: "a", User: "u", Database: "DB"},
		{Account: "a", User: "u", Schema: "SC"},
		{Database: "DB", Schema: "SC"},
	} {
		cfg.PrivateKey = testPEM(t)
		if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "required") {
			t.Errorf("New(%+v): err = %v, want the missing field named", cfg, err)
		}
	}
	if f.count() != 0 {
		t.Errorf("%d requests sent for bad input", f.count())
	}
}

type failingMarshaler struct{}

func (failingMarshaler) MarshalJSON() ([]byte, error) { return nil, errors.New("bad value 555-0100") }

func TestAnEncodeErrorDoesNotQuoteTheValue(t *testing.T) {
	c := newClient(t, &fake{})
	for _, v := range []any{json.Number("555-0100"), failingMarshaler{}} {
		err := appendRows(c, context.Background(), "T", []map[string]any{{"phone": v}})
		if err == nil || strings.Contains(err.Error(), "555-0100") {
			t.Errorf("%T: err = %v, want an error without the value", v, err)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Instrumentation and mocking libraries replace http.DefaultTransport.
func TestNewAcceptsAReplacedDefaultTransport(t *testing.T) {
	orig := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = orig })
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("offline")
	})
	c, err := New(Config{Account: "a", User: "u", PrivateKey: testPEM(t), Database: "DB", Schema: "SC"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Ping(context.Background()); err == nil || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("Ping err = %v, want it to go through the replaced transport", err)
	}
}

func TestAnEmptyIngestHostIsAnErrorAndNotKept(t *testing.T) {
	hosts := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hosts++
	}))
	defer srv.Close()
	c, err := New(Config{Account: "a", User: "u", PrivateKey: testPEM(t), Database: "DB", Schema: "SC", BaseURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := c.Connect(context.Background()); err == nil || !strings.Contains(err.Error(), "empty") {
			t.Fatalf("Connect err = %v, want the empty host reported", err)
		}
	}
	if hosts != 2 {
		t.Errorf("host asked %d times, want 2", hosts)
	}
}

// Snowflake's message can quote a row, and callers log errors, so it never reaches them.
func TestAnAppendErrorCarriesTheCodeButNotSnowflakesMessage(t *testing.T) {
	f := &fake{appendStatus: []int{400}}
	c := newClient(t, f)
	err := appendRows(c, context.Background(), "T", []map[string]any{{"id": "1"}})
	var se *StatusError
	if !errors.As(err, &se) || se.Code != "ERR_TEST" || se.Message != "" || strings.Contains(err.Error(), "row is bad") {
		t.Fatalf("err = %v", err)
	}
}

// pipeStatus fails every append to one pipe with status and accepts the others.
type pipeStatus struct {
	next   http.RoundTripper
	pipe   string
	status int
}

func (p pipeStatus) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.Contains(r.URL.Path, "/pipes/"+p.pipe+"/") {
		return &http.Response{StatusCode: p.status, Body: io.NopCloser(strings.NewReader(`{"code":"ERR_TEST","message":"row is bad"}`)), Header: http.Header{}, Request: r}, nil
	}
	return p.next.RoundTrip(r)
}

// A caller that writes several tables learns which ones failed, how many rows and why, so it
// can name the failed rows by their own ids.
func TestEachFailedPipeIsReportedWithItsRowsAndReason(t *testing.T) {
	f := &fake{}
	c := newClient(t, f)
	c.http.Transport = pipeStatus{next: c.http.Transport, pipe: "BAD", status: http.StatusForbidden}
	err := c.AppendTables(context.Background(), map[string][]map[string]any{
		"good": {{"id": "g1"}},
		"bad":  {{"id": "b1"}, {"id": "b2"}},
	})
	var ae *AppendError
	if !errors.As(err, &ae) || len(ae.Pipes) != 1 || ae.Pipes[0].Pipe != "BAD" || ae.Pipes[0].Rows != 2 {
		t.Fatalf("err = %v; want BAD with 2 rows", err)
	}
	var se *StatusError
	if !errors.As(err, &se) || se.Status != http.StatusForbidden {
		t.Errorf("err = %v, want the 403 reachable with errors.As", err)
	}
	if len(f.stored()["GOOD"]) != 1 {
		t.Errorf("the good table was not stored")
	}
	if strings.Contains(err.Error(), "b1") || strings.Contains(err.Error(), "row is bad") {
		t.Errorf("err quotes a row or Snowflake's message: %v", err)
	}
}

type panicky struct{}

func (panicky) MarshalJSON() ([]byte, error) { panic("secret value") }

// panicTransport panics on the first rows request, like a bug in the HTTP stack.
type panicTransport struct {
	next  http.RoundTripper
	fired atomic.Bool
}

func (p *panicTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.Contains(r.URL.Path, "/rows") && p.fired.CompareAndSwap(false, true) {
		panic("transport bug")
	}
	return p.next.RoundTrip(r)
}

// The client never panics: a panic while encoding or sending becomes an error, the worker keeps
// running, and the next call works.
func TestAPanicBecomesAnErrorAndTheClientKeepsWorking(t *testing.T) {
	f := &fake{}
	c := newClient(t, f)

	err := appendRows(c, context.Background(), "T", []map[string]any{{"id": "1"}, {"id": "2", "v": panicky{}}})
	if err == nil || !strings.Contains(err.Error(), "row 1") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("encode panic: err = %v, want an error naming row 1 and not the value", err)
	}

	c.http.Transport = &panicTransport{next: c.http.Transport}
	if err := appendRows(c, context.Background(), "T", []map[string]any{{"id": "3"}}); err == nil {
		t.Fatal("send panic: want an error")
	}
	if err := appendRows(c, context.Background(), "T", []map[string]any{{"id": "4"}}); err != nil {
		t.Fatalf("after a panic: %v", err)
	}
	if n := c.budget.held(); n != 0 {
		t.Errorf("%d bytes held after the panic", n)
	}
}

func TestAnOversizedRowIsTooLargeAndNothingIsSent(t *testing.T) {
	f := &fake{}
	c := newClient(t, f)
	rows := []map[string]any{{"id": "1"}, {"id": "2", "v": strings.Repeat("x", maxRow)}}
	if err := appendRows(c, context.Background(), "T", rows); !errors.Is(err, ErrTooLarge) || f.count() != 0 {
		t.Fatalf("err = %v, requests = %d; want ErrTooLarge and no request", err, f.count())
	}
}

// Rows that do not compress fill the 4 MB cap fast; every request must stay under it and
// every row arrive once.
func TestRowsThatDoNotCompressAreSplitUnderTheCap(t *testing.T) {
	f := &fake{}
	c := newClient(t, f)
	var rows []map[string]any
	for i := range 3000 {
		noise := make([]byte, 2048)
		rand.Read(noise)
		rows = append(rows, map[string]any{"id": fmt.Sprint(i), "v": hex.EncodeToString(noise)})
	}
	if err := appendRows(c, context.Background(), "T", rows); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range f.stored()["T"] {
		seen[r["id"].(string)] = true
	}
	if len(seen) != len(rows) || len(f.stored()["T"]) != len(rows) {
		t.Fatalf("stored %d rows (%d distinct), want %d", len(f.stored()["T"]), len(seen), len(rows))
	}
	for _, a := range f.appends {
		if a.wire > maxPayload {
			t.Errorf("a request of %d bytes on the wire, over %d", a.wire, maxPayload)
		}
	}
	if len(f.appends) < 3 {
		t.Errorf("%d requests for ~12 MB of incompressible rows", len(f.appends))
	}
}

// Many callers on many pipes: every row arrives exactly once, through a fixed number of
// worker goroutines that does not grow with the callers.
func TestManyCallersShareAFixedNumberOfWorkers(t *testing.T) {
	f := &fake{delay: 10 * time.Millisecond}
	c := newClient(t, f, func(cfg *Config) { cfg.NumWorkers = 4 }) // 2 per pipe
	base := runtime.NumGoroutine()

	const callers, pipes, calls, perCall = 40, 8, 10, 25
	var maxGoroutines atomic.Int64
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
				if n := int64(runtime.NumGoroutine()); n > maxGoroutines.Load() {
					maxGoroutines.Store(n)
				}
			}
		}
	}()
	var wg sync.WaitGroup
	for w := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range calls {
				rows := make([]map[string]any, perCall)
				for j := range rows {
					rows[j] = map[string]any{"id": fmt.Sprintf("%d-%d-%d", w, i, j)}
				}
				if err := appendRows(c, context.Background(), fmt.Sprint("P", (w+i)%pipes), rows); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	close(stop)

	seen := map[string]bool{}
	for _, rows := range f.stored() {
		for _, r := range rows {
			id := r["id"].(string)
			if seen[id] {
				t.Fatalf("row %s stored twice", id)
			}
			seen[id] = true
		}
	}
	if len(seen) != callers*calls*perCall {
		t.Fatalf("stored %d rows, want %d", len(seen), callers*calls*perCall)
	}
	// callers + sampler + workers + a few for HTTP/2 and the test server.
	if limit := int64(base + callers + 1 + 4 + 20); maxGoroutines.Load() > limit {
		t.Errorf("goroutines peaked at %d, want at most %d", maxGoroutines.Load(), limit)
	}
	if p := f.peak.Load(); p > 4 {
		t.Errorf("%d requests in flight, more than the 4 workers", p)
	}
	f.pipePeak.Range(func(k, v any) bool {
		if n := v.(*atomic.Int64).Load(); n > 2 {
			t.Errorf("%v had %d requests in flight, over half of 4 workers", k, n)
		}
		return true
	})
	if n := f.count(); n >= callers*calls/2 {
		t.Errorf("%d requests for %d calls: concurrent calls are not sharing requests", n, callers*calls)
	}
}

// A multi-table write: several tables in one call, with no goroutines of the caller's.
func TestAppendTablesStoresEveryTableOrSendsNothingOnABadRow(t *testing.T) {
	f := &fake{delay: 20 * time.Millisecond}
	c := newClient(t, f, func(cfg *Config) { cfg.NumWorkers = 2 })
	tables := map[string][]map[string]any{
		"A": {{"id": "a1"}, {"id": "a2"}},
		"B": {{"id": "b1"}},
		"C": {},
	}
	if err := c.AppendTables(context.Background(), tables); err != nil {
		t.Fatal(err)
	}
	if f.peak.Load() != 2 {
		t.Errorf("%d requests in flight at once, want the 2 tables sent together", f.peak.Load())
	}
	got := f.stored()
	if len(got["A"]) != 2 || len(got["B"]) != 1 || len(got["C"]) != 0 {
		t.Fatalf("stored = %v", got)
	}

	before := f.count()
	bad := map[string][]map[string]any{
		"A": {{"id": "a3"}},
		"B": {{"id": "b2", "v": strings.Repeat("x", maxRow)}},
	}
	err := c.AppendTables(context.Background(), bad)
	if !errors.Is(err, ErrTooLarge) || !strings.Contains(err.Error(), "B") || f.count() != before {
		t.Fatalf("err = %v, requests %d -> %d; want ErrTooLarge naming B and nothing sent", err, before, f.count())
	}
}

// The default pool is one worker: every request goes out one at a time, and concurrent
// callers still all get their rows stored.
func TestTheDefaultPoolIsOneWorker(t *testing.T) {
	f := &fake{delay: 10 * time.Millisecond}
	c := newClient(t, f)
	var wg sync.WaitGroup
	for w := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := appendRows(c, context.Background(), fmt.Sprint("P", w%4), []map[string]any{{"id": fmt.Sprint(w)}}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if c.cfg.NumWorkers != 1 || f.peak.Load() != 1 {
		t.Fatalf("NumWorkers = %d, peak in flight = %d; want 1, 1", c.cfg.NumWorkers, f.peak.Load())
	}
	n := 0
	for _, rows := range f.stored() {
		n += len(rows)
	}
	if n != 20 {
		t.Fatalf("stored %d rows, want 20", n)
	}
}

// One busy pipe gets at most half the workers, even when more are idle.
func TestOneBusyPipeCannotTakeEveryWorker(t *testing.T) {
	f := &fake{delay: 20 * time.Millisecond}
	c := newClient(t, f, func(cfg *Config) { cfg.NumWorkers = 6 }) // 3 per pipe
	var wg sync.WaitGroup
	for w := range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 5 {
				if err := appendRows(c, context.Background(), "HOT", []map[string]any{{"id": fmt.Sprint(w, "-", i)}}); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if v, ok := f.pipePeak.Load("HOT"); !ok || v.(*atomic.Int64).Load() != 3 {
		t.Fatalf("HOT peaked at %v requests in flight, want 3", v)
	}
}

// peakHeld samples the bytes held by calls in progress until stop is closed.
func peakHeld(c *Client, stop <-chan struct{}) *atomic.Int64 {
	var peak atomic.Int64
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
				if n := int64(c.budget.held()); n > peak.Load() {
					peak.Store(n)
				}
			}
		}
	}()
	return &peak
}

// Rows held across all pipes never pass the cap: a call that would go over waits for room,
// and gives up at its deadline.
func TestQueuedRowsStayUnderTheCapAndCallersWaitForRoom(t *testing.T) {
	f := &fake{delay: 30 * time.Millisecond}
	c := newClient(t, f)
	c.budget.limit = 2 << 10 // a few calls' worth

	stop := make(chan struct{})
	peak := peakHeld(c, stop)
	var ok, waitedOut atomic.Int64
	var wg sync.WaitGroup
	for w := range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()
			rows := []map[string]any{{"id": fmt.Sprint(w), "v": strings.Repeat("x", 400)}}
			switch err := appendRows(c, ctx, fmt.Sprint("P", w%3), rows); {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, context.DeadlineExceeded):
				waitedOut.Add(1)
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	time.Sleep(200 * time.Millisecond) // let workers drop the rows of callers that left
	close(stop)

	if p := peak.Load(); p > 2<<10 {
		t.Errorf("held %d bytes, over the %d cap", p, 2<<10)
	}
	if ok.Load() == 0 || waitedOut.Load() == 0 {
		t.Errorf("%d stored, %d gave up waiting; want some of each", ok.Load(), waitedOut.Load())
	}
	if n := c.budget.held(); n != 0 {
		t.Errorf("%d bytes still held after every call returned", n)
	}
}

// A call bigger than the whole cap still goes through when nothing else is queued.
func TestACallBiggerThanTheCapIsNotStuck(t *testing.T) {
	f := &fake{}
	c := newClient(t, f)
	c.budget.limit = 100
	if err := appendRows(c, context.Background(), "T", []map[string]any{{"v": strings.Repeat("x", 1000)}}); err != nil {
		t.Fatal(err)
	}
}

// During an outage callers get errors at their deadline, memory and goroutines stay bounded,
// nothing is left held afterwards, and the same client works again once Snowflake is back.
func TestAnOutageIsBoundedAndTheClientRecovers(t *testing.T) {
	f := &fake{}
	c := newClient(t, f, func(cfg *Config) { cfg.NumWorkers = 2 })
	if err := appendRows(c, context.Background(), "T", []map[string]any{{"id": "warm"}}); err != nil {
		t.Fatal(err)
	}
	base := runtime.NumGoroutine()
	f.down.Store(true)

	stop := make(chan struct{})
	peak := peakHeld(c, stop)
	var maxGoroutines atomic.Int64
	var failed atomic.Int64
	var wg sync.WaitGroup
	for w := range 10 { // like 10 streams, each retrying its write until the outage ends
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 8; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
				err := appendRows(c, ctx, fmt.Sprint("P", w%3), []map[string]any{{"id": fmt.Sprint(w, "-", i)}})
				cancel()
				if err == nil {
					t.Error("an append stored during the outage")
				}
				failed.Add(1)
				if n := int64(runtime.NumGoroutine()); n > maxGoroutines.Load() {
					maxGoroutines.Store(n)
				}
			}
		}()
	}
	wg.Wait()
	close(stop)

	if limit := int64(base + 10 + 1 + 10); maxGoroutines.Load() > limit {
		t.Errorf("goroutines peaked at %d during the outage, want at most %d", maxGoroutines.Load(), limit)
	}
	if p := peak.Load(); p > maxQueued {
		t.Errorf("held %d bytes during the outage", p)
	}

	f.down.Store(false)
	deadline := time.Now().Add(5 * time.Second)
	for c.budget.held() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := c.budget.held(); n != 0 {
		t.Errorf("%d bytes still held after the outage", n)
	}
	if err := appendRows(c, context.Background(), "T", []map[string]any{{"id": "after"}}); err != nil {
		t.Fatalf("after the outage: %v", err)
	}
}

// A hung token endpoint must not hold callers past their deadline, and a later call works
// once it answers again.
func TestAHungTokenEndpointDoesNotHoldCallers(t *testing.T) {
	f := &fake{tokenGate: make(chan struct{})}
	c := newClient(t, f, func(cfg *Config) { cfg.NumWorkers = 2 })
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			start := time.Now()
			if err := appendRows(c, ctx, "T", []map[string]any{{"id": "1"}}); !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("err = %v", err)
			}
			if took := time.Since(start); took > time.Second {
				t.Errorf("returned after %v, deadline was 200ms", took)
			}
		}()
	}
	wg.Wait()
	close(f.tokenGate)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := appendRows(c, ctx, "T", []map[string]any{{"id": "2"}}); err != nil {
		t.Fatalf("after the token endpoint recovered: %v", err)
	}
}

// A request waiting behind a token refresh gives up at its own deadline, not the refresh's.
func TestWaitingForTheTokenLockEndsAtTheDeadline(t *testing.T) {
	c := newClient(t, &fake{})
	c.auth.lock <- struct{}{} // a refresh in progress that never finishes
	defer func() { <-c.auth.lock }()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, _, err := c.auth.ingestToken(ctx, ""); !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatalf("err = %v after %v", err, time.Since(start))
	}
}

// The checkpoint contract: when an append returns nil the rows are already stored.
func TestAppendReturnsOnlyOnceTheRowsAreStored(t *testing.T) {
	f := &fake{delay: 30 * time.Millisecond}
	c := newClient(t, f)
	for i := 1; i <= 4; i++ {
		if err := appendRows(c, context.Background(), "T", []map[string]any{{"id": fmt.Sprint(i)}}); err != nil {
			t.Fatal(err)
		}
		if n := len(f.stored()["T"]); n != i {
			t.Fatalf("after call %d the server has %d rows", i, n)
		}
	}
}

func TestCloseWaitsForAppendsThenRejectsNewOnes(t *testing.T) {
	f := &fake{delay: 50 * time.Millisecond}
	c := newClient(t, f)
	errc := make(chan error)
	go func() { errc <- appendRows(c, context.Background(), "T", []map[string]any{{"id": "1"}}) }()
	time.Sleep(10 * time.Millisecond)
	if err := c.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-errc; err != nil || len(f.stored()["T"]) != 1 {
		t.Fatalf("in-flight append: err = %v, stored = %d", err, len(f.stored()["T"]))
	}
	if err := appendRows(c, context.Background(), "T", []map[string]any{{"id": "2"}}); !errors.Is(err, ErrClosed) {
		t.Fatalf("after Close: err = %v", err)
	}
}

func TestCloseDeadlineCutsOffRequestsInFlight(t *testing.T) {
	f := &fake{stall: true}
	c := newClient(t, f)
	errc := make(chan error)
	go func() { errc <- appendRows(c, context.Background(), "T", []map[string]any{{"id": "1"}}) }()
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close err = %v", err)
	}
	if err := <-errc; !errors.Is(err, ErrClosed) {
		t.Fatalf("append err = %v, want ErrClosed", err)
	}
}

func TestRetryAfterReadsSeconds(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"3", 3 * time.Second},
		{"-1", 0},
		{"soon", 0},
	}
	for _, tc := range cases {
		if got := retryAfter(tc.in); got != tc.want {
			t.Errorf("retryAfter(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestJWTIsSignedForTheAccountUser(t *testing.T) {
	c, err := New(Config{Account: "xy12345.us-east-1", User: "loader", PrivateKey: testPEM(t), Database: "DB", Schema: "SC"})
	if err != nil {
		t.Fatal(err)
	}
	token, err := c.auth.jwt()
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts", len(parts))
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&testKey.PublicKey, crypto.SHA256, sum[:], sig); err != nil {
		t.Fatalf("signature: %v", err)
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims struct {
		Iss, Sub string
		Iat, Exp int64
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(&testKey.PublicKey)
	fp := sha256.Sum256(der)
	if claims.Sub != "XY12345.LOADER" || claims.Iss != "XY12345.LOADER.SHA256:"+base64.StdEncoding.EncodeToString(fp[:]) {
		t.Errorf("claims = %+v", claims)
	}
	if life := claims.Exp - claims.Iat; life <= 0 || life > 3600 {
		t.Errorf("token life = %ds, want at most an hour", life)
	}
}

func TestBadPrivateKeyFailsAtNew(t *testing.T) {
	for _, key := range []string{"", "not a key", "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n"} {
		if _, err := New(Config{PrivateKey: key}); err == nil {
			t.Errorf("%q: want an error", key)
		}
	}
}

func TestQueryWaitsForARunningStatementAndReadsEveryPartition(t *testing.T) {
	polls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/statements", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Statement string
			Warehouse string
			Bindings  map[string]map[string]string
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Warehouse != "WH" || req.Bindings["1"]["value"] != "SC" {
			t.Errorf("request = %+v, %v", req, err)
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"statementHandle": "h1"}`)
	})
	mux.HandleFunc("GET /api/v2/statements/h1", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("partition") == "1" {
			fmt.Fprint(w, `{"data": [["B"]]}`)
			return
		}
		polls++
		if polls == 1 {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"statementHandle": "h1"}`)
			return
		}
		fmt.Fprint(w, `{"statementHandle": "h1", "resultSetMetaData": {"partitionInfo": [{}, {}]}, "data": [["A"], [null]]}`)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()
	c, err := New(Config{Account: "a", User: "u", PrivateKey: testPEM(t), Database: "DB", Schema: "SC", Warehouse: "WH", BaseURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := c.Query(context.Background(), "SELECT 1 WHERE x = ?", "SC")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || *rows[0][0] != "A" || rows[1][0] != nil || *rows[2][0] != "B" {
		t.Errorf("rows = %v, want A, NULL, B", rows)
	}
}

func TestQueryReportsTheSQLError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req["warehouse"] != nil || req["role"] != nil {
			t.Errorf("request = %v, %v; want no warehouse or role when unset", req, err)
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"code": "002003", "message": "Schema 'SC' does not exist"}`)
	}))
	defer srv.Close()
	c, err := New(Config{Account: "a", User: "u", PrivateKey: testPEM(t), Database: "DB", Schema: "SC", BaseURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Query(context.Background(), "SELECT 1"); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("err = %v, want the SQL error", err)
	}
}
