// Package httpretry provides an http.RoundTripper that retries requests which
// fail with retryable HTTP status codes (5xx and 429) using exponential backoff,
// honoring a server-supplied retry delay when present.
//
// Use NewClient to build a retry-enabled *http.Client, or wrap an existing
// client's Transport with NewTransport. Either way requests become resilient
// to transient upstream failures without changing call sites.
package httpretry

import (
	"context"
	"io"
	"net/http"
	"time"
)

const (
	// defaultMaxRetries is the number of retries attempted after the first try.
	defaultMaxRetries = 3
	// defaultBaseDelay is the wait before the first retry; it doubles each retry.
	defaultBaseDelay = 1 * time.Second
	// defaultMaxDelay caps the per-attempt backoff wait.
	defaultMaxDelay = 8 * time.Second
	// maxServerDelay is the longest server-instructed retry wait we honor.
	// A longer instruction is treated as unrecoverable within a single call
	// and returned to the caller instead of retried (see ADR-0101).
	maxServerDelay = 60 * time.Second
)

// Transport is an http.RoundTripper that retries retryable responses
// (HTTP 5xx and 429) with exponential backoff. On a 429 it honors a
// server-supplied retry delay (Retry-After header, then the response body's
// RetryInfo) in place of the backoff when present. The backoff parameters
// are fixed at construction time via NewTransport.
type Transport struct {
	base       http.RoundTripper
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	// perAttemptTimeout, when non-zero, bounds each individual attempt via
	// its own context so a long server-instructed wait between attempts
	// doesn't count against it. Set by NewClient.
	perAttemptTimeout time.Duration
}

// NewTransport wraps base with retry logic using fixed backoff settings.
// If base is nil, http.DefaultTransport is used.
func NewTransport(base http.RoundTripper) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &Transport{
		base:       base,
		maxRetries: defaultMaxRetries,
		baseDelay:  defaultBaseDelay,
		maxDelay:   defaultMaxDelay,
	}
}

// NewClient returns an *http.Client whose Transport retries 5xx/429 responses
// with exponential backoff (or a server-instructed delay, see ADR-0101).
// timeout, if > 0, bounds each individual attempt rather than the whole
// call, since the whole call may legitimately include a long retry wait.
func NewClient(timeout time.Duration) *http.Client {
	tr := NewTransport(nil)
	tr.perAttemptTimeout = timeout
	return &http.Client{Transport: tr}
}

// RoundTrip implements http.RoundTripper. It retries the request while the
// response status is retryable and attempts remain, waiting between tries.
// On 429 it prefers a server-instructed delay over the exponential backoff;
// a non-nil transport error is returned immediately (the underlying
// transport already handles connection-level retries).
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var delay time.Duration
	for attempt := 0; ; attempt++ {
		reqToSend := req
		if attempt > 0 {
			if err := t.wait(req, delay); err != nil {
				return nil, err
			}
			clone, err := cloneRequest(req)
			if err != nil {
				return nil, err
			}
			reqToSend = clone
		}

		resp, err := t.roundTripAttempt(reqToSend)
		if err != nil {
			return nil, err
		}

		if !isRetryable(resp.StatusCode) || attempt >= t.maxRetries {
			return resp, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			if serverDelay, ok := extractServerDelay(resp); ok {
				if serverDelay > maxServerDelay {
					return resp, nil
				}
				delay = serverDelay
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				continue
			}
		}

		delay = t.backoff(attempt + 1)
		// Will retry: drain and close the body so the connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

// roundTripAttempt performs a single attempt, applying perAttemptTimeout to
// its own context when set. The returned response's body cancels that
// context on Close, once the caller (or the retry loop) is done with it.
func (t *Transport) roundTripAttempt(req *http.Request) (*http.Response, error) {
	if t.perAttemptTimeout <= 0 {
		return t.base.RoundTrip(req)
	}

	ctx, cancel := context.WithTimeout(req.Context(), t.perAttemptTimeout)
	resp, err := t.base.RoundTrip(req.WithContext(ctx))
	if err != nil {
		cancel()
		return nil, err
	}
	resp.Body = &cancelOnCloseBody{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

// cancelOnCloseBody cancels an attempt's per-attempt context once its body
// is closed, so the context is released whether the response is consumed
// by the caller or drained by the retry loop.
type cancelOnCloseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelOnCloseBody) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

// wait sleeps for delay, returning early with the context error if the
// request context is cancelled.
func (t *Transport) wait(req *http.Request, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-req.Context().Done():
		return req.Context().Err()
	}
}

// backoff returns the wait before the given attempt (attempt >= 1):
// baseDelay, 2*baseDelay, 4*baseDelay, ... capped at maxDelay.
func (t *Transport) backoff(attempt int) time.Duration {
	delay := t.baseDelay << (attempt - 1)
	if delay <= 0 || delay > t.maxDelay {
		delay = t.maxDelay
	}
	return delay
}

// isRetryable reports whether an HTTP status code should be retried.
func isRetryable(code int) bool {
	return code == http.StatusTooManyRequests || (code >= 500 && code <= 599)
}

// cloneRequest returns a copy of req with a fresh body suitable for retrying.
// Requests without a re-readable body (no body, or no GetBody) are returned
// as-is, which means they are effectively attempted only once if they fail.
func cloneRequest(req *http.Request) (*http.Request, error) {
	if req.Body == nil || req.Body == http.NoBody || req.GetBody == nil {
		return req, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.Body = body
	return clone, nil
}
