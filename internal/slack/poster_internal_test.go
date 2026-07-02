package slack

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	slackgo "github.com/slack-go/slack"
)

func writeTempAudioFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.mp3")
	if err := os.WriteFile(path, []byte("fake mp3 content"), 0o644); err != nil {
		t.Fatalf("write temp audio: %v", err)
	}
	return path
}

func newUploadTestServer(t *testing.T, filesInfoHandler func(n int32) any) *httptest.Server {
	t.Helper()
	var callCount int32
	var srvURL string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "files.getUploadURLExternal"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":         true,
				"upload_url": srvURL + "/upload",
				"file_id":    "FTEST123",
			})
		case r.URL.Path == "/upload":
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "files.completeUploadExternal"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":    true,
				"files": []map[string]any{{"id": "FTEST123"}},
			})
		case strings.HasSuffix(r.URL.Path, "files.info"):
			n := atomic.AddInt32(&callCount, 1)
			_ = json.NewEncoder(w).Encode(filesInfoHandler(n))
		default:
			http.NotFound(w, r)
		}
	}))
	srvURL = srv.URL
	return srv
}

// newFilesInfoServer creates a test server that only handles files.info requests.
func newFilesInfoServer(t *testing.T, filesInfoHandler func(n int32) any) *httptest.Server {
	t.Helper()
	var callCount int32

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "files.info") {
			n := atomic.AddInt32(&callCount, 1)
			_ = json.NewEncoder(w).Encode(filesInfoHandler(n))
		} else {
			http.NotFound(w, r)
		}
	}))
}

func filesInfoResp(channel, ts string, isPrivate bool) map[string]any {
	public := map[string]any{}
	private := map[string]any{}
	if ts != "" {
		entry := []map[string]any{{"ts": ts}}
		if isPrivate {
			private[channel] = entry
		} else {
			public[channel] = entry
		}
	}
	return map[string]any{
		"ok": true,
		"file": map[string]any{
			"id":     "FTEST123",
			"shares": map[string]any{"public": public, "private": private},
		},
		"paging": map[string]any{"count": 1, "total": 1, "page": 1, "pages": 1},
	}
}

func newTestRealPoster(apiURL string, pollInterval, pollTimeout time.Duration) *realPoster {
	return &realPoster{
		client:       slackgo.New("xoxb-test", slackgo.OptionAPIURL(apiURL)),
		pollInterval: pollInterval,
		pollTimeout:  pollTimeout,
	}
}

func TestRealPoster_UploadAudio_ReturnsFileID(t *testing.T) {
	srv := newUploadTestServer(t, func(_ int32) any {
		// files.info not called during UploadAudio
		return nil
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 5*time.Second)
	filePath := writeTempAudioFile(t)

	fileID, err := poster.UploadAudio(context.Background(), UploadParams{
		Channel:  "C0123456789",
		FilePath: filePath,
		Filename: "test.mp3",
	})

	if err != nil {
		t.Fatalf("UploadAudio should succeed: %v", err)
	}
	if fileID != "FTEST123" {
		t.Errorf("fileID = %q, want %q", fileID, "FTEST123")
	}
}

func TestRealPoster_ResolveThreadTS_SuccessOnNthPoll(t *testing.T) {
	const (
		channel = "C0123456789"
		wantTS  = "1234567890.123456"
	)

	srv := newFilesInfoServer(t, func(n int32) any {
		if n < 3 {
			return filesInfoResp(channel, "", false)
		}
		return filesInfoResp(channel, wantTS, false)
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 5*time.Second)

	ts, err := poster.ResolveThreadTS(context.Background(), "FTEST123", channel)

	if err != nil {
		t.Fatalf("ResolveThreadTS should succeed after retries: %v", err)
	}
	if ts != wantTS {
		t.Errorf("ts = %q, want %q", ts, wantTS)
	}
}

func TestRealPoster_ResolveThreadTS_PrivateChannel_Success(t *testing.T) {
	const (
		channel = "C9876543210"
		wantTS  = "9999999999.999999"
	)

	srv := newFilesInfoServer(t, func(_ int32) any {
		return filesInfoResp(channel, wantTS, true)
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 5*time.Second)

	ts, err := poster.ResolveThreadTS(context.Background(), "FTEST123", channel)

	if err != nil {
		t.Fatalf("ResolveThreadTS should succeed for private channel: %v", err)
	}
	if ts != wantTS {
		t.Errorf("ts = %q, want %q", ts, wantTS)
	}
}

func TestRealPoster_ResolveThreadTS_PollTimeout_ReturnsError(t *testing.T) {
	const channel = "C0123456789"

	srv := newFilesInfoServer(t, func(_ int32) any {
		return filesInfoResp(channel, "", false)
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 100*time.Millisecond)

	_, err := poster.ResolveThreadTS(context.Background(), "FTEST123", channel)

	if err == nil {
		t.Fatal("ResolveThreadTS should return error on poll timeout")
	}
	if !strings.Contains(err.Error(), "FTEST123") {
		t.Errorf("error should contain fileID, got: %v", err)
	}
	if !strings.Contains(err.Error(), "二重投稿") {
		t.Errorf("error should mention double-posting, got: %v", err)
	}
}

func TestRealPoster_ResolveThreadTS_RetryableError_WrapsOnTimeout(t *testing.T) {
	const channel = "C0123456789"

	srv := newFilesInfoServer(t, func(_ int32) any {
		return map[string]any{"ok": false, "error": "service_error"}
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 100*time.Millisecond)

	_, err := poster.ResolveThreadTS(context.Background(), "FTEST123", channel)

	if err == nil {
		t.Fatal("ResolveThreadTS should return error on poll timeout")
	}
	if !strings.Contains(err.Error(), "FTEST123") {
		t.Errorf("error should contain fileID, got: %v", err)
	}
	if !strings.Contains(err.Error(), "二重投稿") {
		t.Errorf("error should mention double-posting, got: %v", err)
	}
	if !strings.Contains(err.Error(), "service_error") {
		t.Errorf("error should wrap last GetFileInfo error, got: %v", err)
	}
}

func TestRealPoster_ResolveThreadTS_NonRetryableError_ImmediateFailure(t *testing.T) {
	const channel = "C0123456789"

	var infoCallCount int32
	srv := newFilesInfoServer(t, func(n int32) any {
		atomic.StoreInt32(&infoCallCount, n)
		return map[string]any{"ok": false, "error": "missing_scope"}
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 200*time.Millisecond)

	_, err := poster.ResolveThreadTS(context.Background(), "FTEST123", channel)

	if err == nil {
		t.Fatal("ResolveThreadTS should return error for non-retryable Slack error")
	}
	if !strings.Contains(err.Error(), "FTEST123") {
		t.Errorf("error should contain fileID, got: %v", err)
	}
	if !strings.Contains(err.Error(), "二重投稿") {
		t.Errorf("error should mention double-posting, got: %v", err)
	}
	if !strings.Contains(err.Error(), "missing_scope") {
		t.Errorf("error should wrap original Slack error, got: %v", err)
	}
	if n := atomic.LoadInt32(&infoCallCount); n != 1 {
		t.Errorf("files.info should be called exactly once for non-retryable error, got %d calls", n)
	}
}

// newAuthTestServer creates a test server that handles auth.test, optionally
// setting the X-OAuth-Scopes header. When ok is false it returns invalid_auth.
func newAuthTestServer(t *testing.T, scopesHeader string, ok bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "auth.test") {
			http.NotFound(w, r)
			return
		}
		if scopesHeader != "" {
			w.Header().Set("X-OAuth-Scopes", scopesHeader)
		}
		w.Header().Set("Content-Type", "application/json")
		if ok {
			_, _ = w.Write([]byte(`{"ok":true,"url":"https://x.slack.com/","team":"T","user":"u","team_id":"T1","user_id":"U1","bot_id":"B1"}`))
		} else {
			_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
		}
	}))
}

func TestRealPoster_VerifyScopes_AllPresent_Succeeds(t *testing.T) {
	srv := newAuthTestServer(t, "chat:write,files:write,files:read", true)
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 5*time.Second)

	err := poster.VerifyScopes(context.Background(), []string{"files:write", "files:read", "chat:write"})
	if err != nil {
		t.Fatalf("VerifyScopes should succeed when all scopes are granted: %v", err)
	}
}

func TestRealPoster_VerifyScopes_MissingScope_ReturnsError(t *testing.T) {
	srv := newAuthTestServer(t, "chat:write,files:write", true)
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 5*time.Second)

	err := poster.VerifyScopes(context.Background(), []string{"files:write", "files:read", "chat:write"})
	if err == nil {
		t.Fatal("VerifyScopes should return error when a required scope is missing")
	}
	if !strings.Contains(err.Error(), "files:read") {
		t.Errorf("error should name the missing scope files:read, got: %v", err)
	}
}

func TestRealPoster_VerifyScopes_AuthError_ReturnsError(t *testing.T) {
	srv := newAuthTestServer(t, "", false)
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 5*time.Second)

	err := poster.VerifyScopes(context.Background(), []string{"files:write"})
	if err == nil {
		t.Fatal("VerifyScopes should return error when auth.test fails")
	}
	if !strings.Contains(err.Error(), "認証") {
		t.Errorf("error should indicate auth failure, got: %v", err)
	}
}

func TestRealPoster_VerifyScopes_EmptyHeader_SkipsCheck(t *testing.T) {
	srv := newAuthTestServer(t, "", true)
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 5*time.Second)

	// スコープヘッダが無い場合は判定不能としてスキップし、エラーにしない。
	err := poster.VerifyScopes(context.Background(), []string{"files:write", "files:read"})
	if err != nil {
		t.Fatalf("VerifyScopes should skip check when scope header is absent: %v", err)
	}
}

func TestIsNonRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"missing_scope", slackgo.SlackErrorResponse{Err: "missing_scope"}, true},
		{"not_authed", slackgo.SlackErrorResponse{Err: "not_authed"}, true},
		{"invalid_auth", slackgo.SlackErrorResponse{Err: "invalid_auth"}, true},
		{"account_inactive", slackgo.SlackErrorResponse{Err: "account_inactive"}, true},
		{"token_revoked", slackgo.SlackErrorResponse{Err: "token_revoked"}, true},
		{"token_expired", slackgo.SlackErrorResponse{Err: "token_expired"}, true},
		{"no_permission", slackgo.SlackErrorResponse{Err: "no_permission"}, true},
		{"file_not_found", slackgo.SlackErrorResponse{Err: "file_not_found"}, true},
		{"file_deleted", slackgo.SlackErrorResponse{Err: "file_deleted"}, true},
		{"unknown Slack code is retryable", slackgo.SlackErrorResponse{Err: "service_error"}, false},
		{"non-Slack error is retryable", errors.New("connection refused"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNonRetryable(tt.err); got != tt.want {
				t.Errorf("isNonRetryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
