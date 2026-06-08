// Package httpretry provides an http.RoundTripper that retries requests which
// fail with retryable HTTP status codes (5xx and 429) using exponential backoff.
//
// Wrap any http.Client's Transport with NewTransport to make its requests
// resilient to transient upstream failures without changing call sites.
package httpretry

import (
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
)

// Transport is an http.RoundTripper that retries retryable responses
// (HTTP 5xx and 429) with exponential backoff. The backoff parameters are
// fixed at construction time via NewTransport.
type Transport struct {
	base       http.RoundTripper
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
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

// RoundTrip implements http.RoundTripper. It retries the request while the
// response status is retryable and attempts remain, waiting with exponential
// backoff between tries. A non-nil transport error is returned immediately
// (the underlying transport already handles connection-level retries).
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			if err := t.wait(req, attempt); err != nil {
				return nil, err
			}
		}

		reqCopy, err := cloneRequest(req)
		if err != nil {
			return nil, err
		}

		resp, err := t.base.RoundTrip(reqCopy)
		if err != nil {
			return nil, err
		}

		if !isRetryable(resp.StatusCode) || attempt >= t.maxRetries {
			return resp, nil
		}

		// Will retry: drain and close the body so the connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

// wait sleeps for the backoff duration of the given attempt, returning early
// with the context error if the request context is cancelled.
func (t *Transport) wait(req *http.Request, attempt int) error {
	timer := time.NewTimer(t.backoff(attempt))
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
