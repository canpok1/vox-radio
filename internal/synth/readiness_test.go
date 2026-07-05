package synth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

func TestCheckReadiness_ReturnsNilWhenEngineReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`"0.14.0"`))
	}))
	defer server.Close()

	one := 1
	cfg := &config.Config{Voicevox: config.VoicevoxConfig{StartupTimeoutSeconds: &one}}
	if err := CheckReadiness(context.Background(), map[string]string{"default": server.URL}, cfg); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckReadiness_SkipsWhenStartupTimeoutZero(t *testing.T) {
	zero := 0
	cfg := &config.Config{Voicevox: config.VoicevoxConfig{StartupTimeoutSeconds: &zero}}
	// URL is unreachable, but a zero timeout disables the check so no connection is attempted.
	if err := CheckReadiness(context.Background(), map[string]string{"default": "http://127.0.0.1:0"}, cfg); err != nil {
		t.Fatalf("expected nil (check disabled), got %v", err)
	}
}

func TestCheckReadiness_ReturnsErrorWhenUnreachable(t *testing.T) {
	// Start then immediately close a server so the URL refuses connections.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	one := 1
	cfg := &config.Config{Voicevox: config.VoicevoxConfig{StartupTimeoutSeconds: &one}}
	err := CheckReadiness(context.Background(), map[string]string{"default": url}, cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "VOICEVOX") {
		t.Errorf("want 'VOICEVOX' in error, got: %v", err)
	}
}

func TestCheckReadiness_ChecksAllEnginesConcurrently_AndNamesFailingOnes(t *testing.T) {
	ready := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`"0.14.0"`))
	}))
	defer ready.Close()

	unreachable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachableURL := unreachable.URL
	unreachable.Close()

	one := 1
	cfg := &config.Config{Voicevox: config.VoicevoxConfig{StartupTimeoutSeconds: &one}}
	err := CheckReadiness(context.Background(), map[string]string{
		"default": ready.URL,
		"nemo":    unreachableURL,
	}, cfg)
	if err == nil {
		t.Fatal("expected error when one of multiple engines is unreachable")
	}
	if !strings.Contains(err.Error(), "nemo") {
		t.Errorf("error should name the failing engine (nemo), got: %v", err)
	}
	if !strings.Contains(err.Error(), unreachableURL) {
		t.Errorf("error should include the failing engine's URL, got: %v", err)
	}
}
