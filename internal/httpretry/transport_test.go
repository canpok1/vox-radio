package httpretry

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newFastTransport returns a Transport with near-zero backoff so tests run fast.
func newFastTransport() *Transport {
	tr := NewTransport(nil)
	tr.baseDelay = time.Millisecond
	tr.maxDelay = 2 * time.Millisecond
	return tr
}

func TestRoundTrip_RetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	client := &http.Client{Transport: newFastTransport()}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("calls = %d, want 3", got)
	}
}

func TestRoundTrip_RetriesOn429ThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: newFastTransport()}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
}

func TestRoundTrip_ExhaustsRetriesAndReturnsLastResponse(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := &http.Client{Transport: newFastTransport()}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
	// 1 initial attempt + defaultMaxRetries retries.
	if got, want := atomic.LoadInt32(&calls), int32(defaultMaxRetries+1); got != want {
		t.Fatalf("calls = %d, want %d", got, want)
	}
}

func TestRoundTrip_NoRetryOnNon5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client := &http.Client{Transport: newFastTransport()}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls = %d, want 1 (no retry on 4xx)", got)
	}
}

func TestRoundTrip_PostBodyResentOnRetry(t *testing.T) {
	var (
		mu     sync.Mutex
		bodies []string
		calls  int32
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		mu.Unlock()
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: newFastTransport()}
	resp, err := client.Post(srv.URL, "text/plain", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 3 {
		t.Fatalf("received %d requests, want 3", len(bodies))
	}
	for i, b := range bodies {
		if b != "payload" {
			t.Fatalf("request %d body = %q, want %q", i, b, "payload")
		}
	}
}

func TestRoundTrip_ContextCancelledDuringBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	tr := NewTransport(nil)
	tr.baseDelay = 100 * time.Millisecond
	tr.maxDelay = 100 * time.Millisecond
	client := &http.Client{Transport: tr}

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	// Should abort during the first backoff, well before the full 100ms.
	if elapsed := time.Since(start); elapsed > 80*time.Millisecond {
		t.Fatalf("took %v, expected early cancellation", elapsed)
	}
}

func TestRoundTrip_HonorsRetryAfterHeaderSeconds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewTransport(nil)
	// Backoff is deliberately slow; a fast response proves the header (not
	// the backoff) governed the wait.
	tr.baseDelay = time.Second
	tr.maxDelay = time.Second

	start := time.Now()
	client := &http.Client{Transport: tr}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("took %v, want well under the 1s backoff (Retry-After: 0 should have governed the wait)", elapsed)
	}
}

func TestRoundTrip_HonorsRetryAfterHeaderHTTPDate(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", time.Now().Add(50*time.Millisecond).UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewTransport(nil)
	tr.baseDelay = time.Second
	tr.maxDelay = time.Second

	start := time.Now()
	client := &http.Client{Transport: tr}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("took %v, want well under the 1s backoff (Retry-After date should have governed the wait)", elapsed)
	}
}

func TestRoundTrip_HonorsRetryInfoInBody(t *testing.T) {
	const body = `{"error":{"code":429,"message":"rate limited","details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"0.05s"}]}}`

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, body)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewTransport(nil)
	tr.baseDelay = time.Second
	tr.maxDelay = time.Second

	start := time.Now()
	client := &http.Client{Transport: tr}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("took %v, want well under the 1s backoff (body RetryInfo should have governed the wait)", elapsed)
	}
}

func TestRoundTrip_HonorsRetryInfoInArrayWrappedBody(t *testing.T) {
	const body = `[{"error":{"code":429,"message":"rate limited","details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"0.05s"}]}}]`

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, body)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewTransport(nil)
	tr.baseDelay = time.Second
	tr.maxDelay = time.Second

	client := &http.Client{Transport: tr}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
}

func TestRoundTrip_DoesNotRetryWhenServerDelayExceedsLimit(t *testing.T) {
	const body = `{"error":{"code":429,"message":"rate limited","details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"61s"}]}}`

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	client := &http.Client{Transport: newFastTransport()}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls = %d, want 1 (no retry once the instructed delay exceeds the limit)", got)
	}

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != body {
		t.Fatalf("body = %q, want %q (body must be restored for the caller)", got, body)
	}
}

func TestRoundTrip_PerAttemptTimeoutAppliesToEachAttempt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(10 * time.Millisecond)
	c.Transport.(*Transport).maxRetries = 0 // isolate a single attempt

	_, err := c.Get(srv.URL)
	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestRoundTrip_PerAttemptTimeoutExcludesRetryWait(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewClient(80 * time.Millisecond)
	tr := c.Transport.(*Transport)
	tr.baseDelay = 60 * time.Millisecond
	tr.maxDelay = 60 * time.Millisecond
	tr.maxRetries = 2

	start := time.Now()
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("calls = %d, want 3", got)
	}
	// Total retry wait (2 * 60ms = 120ms) exceeds perAttemptTimeout (80ms);
	// this must still succeed because the timeout bounds each attempt, not the whole call.
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("elapsed = %v, want >= 100ms (retry waits should not be cut short)", elapsed)
	}
}

func TestNewClient_AttachesRetryTransportWithPerAttemptTimeout(t *testing.T) {
	c := NewClient(5 * time.Second)
	if c.Timeout != 0 {
		t.Fatalf("client.Timeout = %v, want 0 (timeout applies per attempt, not to the whole call)", c.Timeout)
	}
	tr, ok := c.Transport.(*Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *Transport", c.Transport)
	}
	if tr.perAttemptTimeout != 5*time.Second {
		t.Fatalf("perAttemptTimeout = %v, want 5s", tr.perAttemptTimeout)
	}

	c0 := NewClient(0)
	if got := c0.Transport.(*Transport).perAttemptTimeout; got != 0 {
		t.Fatalf("perAttemptTimeout = %v, want 0 (unset)", got)
	}
}

func TestIsRetryable(t *testing.T) {
	cases := map[int]bool{
		http.StatusOK:                  false,
		http.StatusBadRequest:          false,
		http.StatusTooManyRequests:     true,
		http.StatusInternalServerError: true,
		http.StatusBadGateway:          true,
		http.StatusServiceUnavailable:  true,
		http.StatusGatewayTimeout:      true,
	}
	for code, want := range cases {
		if got := isRetryable(code); got != want {
			t.Errorf("isRetryable(%d) = %v, want %v", code, got, want)
		}
	}
}

func TestBackoff_ExponentialCappedAtMaxDelay(t *testing.T) {
	tr := NewTransport(nil) // baseDelay=1s, maxDelay=8s
	cases := map[int]time.Duration{
		1: 1 * time.Second,
		2: 2 * time.Second,
		3: 4 * time.Second,
		4: 8 * time.Second,
		5: 8 * time.Second, // capped
	}
	for attempt, want := range cases {
		if got := tr.backoff(attempt); got != want {
			t.Errorf("backoff(%d) = %v, want %v", attempt, got, want)
		}
	}
}
