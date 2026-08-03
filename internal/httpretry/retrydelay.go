package httpretry

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// maxRetryInfoBodyBytes caps how much of a 429 response body is read while
// looking for a RetryInfo delay, so a huge or unbounded body can't be forced
// into memory.
const maxRetryInfoBodyBytes = 64 * 1024

// retryInfoType is the Google RetryInfo detail's @type discriminator.
const retryInfoType = "type.googleapis.com/google.rpc.RetryInfo"

// extractServerDelay reads a server-instructed retry delay from resp: the
// Retry-After header takes priority, then the response body's Google
// RetryInfo detail. If the body must be read to look for RetryInfo, it is
// restored onto resp.Body afterward so the caller can still read it.
// Returns ok=false if no instruction was found.
func extractServerDelay(resp *http.Response) (time.Duration, bool) {
	if d, ok := retryDelayFromHeader(resp.Header.Get("Retry-After")); ok {
		return d, true
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxRetryInfoBodyBytes))
	// Drain any remainder so the underlying connection can be reused; Close
	// alone would abandon it if the body exceeded maxRetryInfoBodyBytes.
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return retryDelayFromBody(body)
}

// retryDelayFromHeader parses a Retry-After header value, which is either a
// number of seconds or an HTTP-date.
func retryDelayFromHeader(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d, true
		}
		return 0, true
	}
	return 0, false
}

// retryInfoDetail is one entry of a Google API error's details[] array.
type retryInfoDetail struct {
	Type       string `json:"@type"`
	RetryDelay string `json:"retryDelay"`
}

// googleErrorBody is a Google API error response, e.g. Gemini's.
type googleErrorBody struct {
	Error struct {
		Details []retryInfoDetail `json:"details"`
	} `json:"error"`
}

// retryDelayFromBody looks for a Google RetryInfo detail in a 429 response
// body. Both the plain object form ({"error": {...}}) and the array-wrapped
// form ([{"error": {...}}], used by Gemini's OpenAI-compatible endpoint) are
// supported.
func retryDelayFromBody(body []byte) (time.Duration, bool) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return 0, false
	}

	var details []retryInfoDetail
	if trimmed[0] == '[' {
		var wrapped []googleErrorBody
		if err := json.Unmarshal(trimmed, &wrapped); err != nil || len(wrapped) == 0 {
			return 0, false
		}
		details = wrapped[0].Error.Details
	} else {
		var parsed googleErrorBody
		if err := json.Unmarshal(trimmed, &parsed); err != nil {
			return 0, false
		}
		details = parsed.Error.Details
	}

	for _, d := range details {
		if d.Type != retryInfoType {
			continue
		}
		if dur, err := time.ParseDuration(d.RetryDelay); err == nil && dur >= 0 {
			return dur, true
		}
	}
	return 0, false
}
