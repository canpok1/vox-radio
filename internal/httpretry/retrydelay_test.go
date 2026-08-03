package httpretry

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryDelayFromHeader(t *testing.T) {
	cases := []struct {
		name      string
		value     string
		wantOK    bool
		wantDelay time.Duration
	}{
		{"empty", "", false, 0},
		{"seconds", "51", true, 51 * time.Second},
		{"zero seconds", "0", true, 0},
		{"negative seconds", "-1", false, 0},
		{"garbage", "not-a-value", false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := retryDelayFromHeader(tc.value)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && d != tc.wantDelay {
				t.Fatalf("delay = %v, want %v", d, tc.wantDelay)
			}
		})
	}
}

func TestRetryDelayFromHeader_HTTPDateInPast(t *testing.T) {
	past := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	d, ok := retryDelayFromHeader(past)
	if !ok {
		t.Fatal("expected ok=true for a past HTTP-date (treated as no wait)")
	}
	if d != 0 {
		t.Fatalf("delay = %v, want 0 for a past date", d)
	}
}

func TestRetryDelayFromBody(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantOK    bool
		wantDelay time.Duration
	}{
		{
			name:      "object form",
			body:      `{"error":{"details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"51s"}]}}`,
			wantOK:    true,
			wantDelay: 51 * time.Second,
		},
		{
			name:      "array-wrapped form",
			body:      `[{"error":{"details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"31.5s"}]}}]`,
			wantOK:    true,
			wantDelay: 31500 * time.Millisecond,
		},
		{
			name:   "no matching detail type",
			body:   `{"error":{"details":[{"@type":"type.googleapis.com/other.Thing","retryDelay":"51s"}]}}`,
			wantOK: false,
		},
		{
			name:   "empty body",
			body:   ``,
			wantOK: false,
		},
		{
			name:   "not json",
			body:   `plain text error`,
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := retryDelayFromBody([]byte(tc.body))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && d != tc.wantDelay {
				t.Fatalf("delay = %v, want %v", d, tc.wantDelay)
			}
		})
	}
}
