package slack

import (
	"context"
	"encoding/json"
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

func TestRealPoster_UploadAudio_SuccessOnNthPoll(t *testing.T) {
	const (
		channel = "C0123456789"
		wantTS  = "1234567890.123456"
	)

	srv := newUploadTestServer(t, func(n int32) any {
		if n < 3 {
			return filesInfoResp(channel, "", false)
		}
		return filesInfoResp(channel, wantTS, false)
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 5*time.Second)
	filePath := writeTempAudioFile(t)

	fileID, ts, err := poster.UploadAudio(context.Background(), UploadParams{
		Channel:  channel,
		FilePath: filePath,
		Filename: "test.mp3",
	})

	if err != nil {
		t.Fatalf("UploadAudio should succeed after retries: %v", err)
	}
	if fileID != "FTEST123" {
		t.Errorf("fileID = %q, want %q", fileID, "FTEST123")
	}
	if ts != wantTS {
		t.Errorf("ts = %q, want %q", ts, wantTS)
	}
}

func TestRealPoster_UploadAudio_PrivateChannel_Success(t *testing.T) {
	const (
		channel = "C9876543210"
		wantTS  = "9999999999.999999"
	)

	srv := newUploadTestServer(t, func(_ int32) any {
		return filesInfoResp(channel, wantTS, true)
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 5*time.Second)
	filePath := writeTempAudioFile(t)

	_, ts, err := poster.UploadAudio(context.Background(), UploadParams{
		Channel:  channel,
		FilePath: filePath,
		Filename: "test.mp3",
	})

	if err != nil {
		t.Fatalf("UploadAudio should succeed for private channel: %v", err)
	}
	if ts != wantTS {
		t.Errorf("ts = %q, want %q", ts, wantTS)
	}
}

func TestRealPoster_UploadAudio_PollTimeout_ReturnsError(t *testing.T) {
	const channel = "C0123456789"

	srv := newUploadTestServer(t, func(_ int32) any {
		return filesInfoResp(channel, "", false)
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 100*time.Millisecond)
	filePath := writeTempAudioFile(t)

	_, _, err := poster.UploadAudio(context.Background(), UploadParams{
		Channel:  channel,
		FilePath: filePath,
		Filename: "test.mp3",
	})

	if err == nil {
		t.Fatal("UploadAudio should return error on poll timeout")
	}
	if !strings.Contains(err.Error(), "FTEST123") {
		t.Errorf("error should contain fileID, got: %v", err)
	}
	if !strings.Contains(err.Error(), "二重投稿") {
		t.Errorf("error should mention double-posting, got: %v", err)
	}
}

func TestRealPoster_UploadAudio_RetryableError_WrapsOnTimeout(t *testing.T) {
	const channel = "C0123456789"

	// service_error is not in the non-retryable list — polling should continue until timeout.
	srv := newUploadTestServer(t, func(_ int32) any {
		return map[string]any{"ok": false, "error": "service_error"}
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 100*time.Millisecond)
	filePath := writeTempAudioFile(t)

	_, _, err := poster.UploadAudio(context.Background(), UploadParams{
		Channel:  channel,
		FilePath: filePath,
		Filename: "test.mp3",
	})

	if err == nil {
		t.Fatal("UploadAudio should return error on poll timeout")
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

func TestRealPoster_UploadAudio_NonRetryableError_ImmediateFailure(t *testing.T) {
	const channel = "C0123456789"

	var infoCallCount int32
	srv := newUploadTestServer(t, func(n int32) any {
		atomic.StoreInt32(&infoCallCount, n)
		return map[string]any{"ok": false, "error": "missing_scope"}
	})
	defer srv.Close()

	poster := newTestRealPoster(srv.URL+"/", 10*time.Millisecond, 200*time.Millisecond)
	filePath := writeTempAudioFile(t)

	_, _, err := poster.UploadAudio(context.Background(), UploadParams{
		Channel:  channel,
		FilePath: filePath,
		Filename: "test.mp3",
	})

	if err == nil {
		t.Fatal("UploadAudio should return error for non-retryable Slack error")
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
