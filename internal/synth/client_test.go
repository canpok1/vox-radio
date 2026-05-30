package synth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPVoicevoxClient_AudioQuery_SendsCorrectRequest(t *testing.T) {
	var gotMethod, gotPath, gotText, gotSpeaker string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotText = r.URL.Query().Get("text")
		gotSpeaker = r.URL.Query().Get("speaker")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AudioQuery{SpeedScale: 1.0, PitchScale: 0.0, IntonationScale: 1.0, VolumeScale: 1.0})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.AudioQuery(context.Background(), "テスト", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", gotMethod)
	}
	if gotPath != "/audio_query" {
		t.Errorf("path: got %s, want /audio_query", gotPath)
	}
	if gotText != "テスト" {
		t.Errorf("text: got %s, want テスト", gotText)
	}
	if gotSpeaker != "3" {
		t.Errorf("speaker: got %s, want 3", gotSpeaker)
	}
}

func TestHTTPVoicevoxClient_AudioQuery_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AudioQuery{
			SpeedScale:        1.2,
			PitchScale:        0.1,
			IntonationScale:   0.8,
			VolumeScale:       1.5,
			PrePhonemeLength:  0.05,
			PostPhonemeLength: 0.05,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	query, err := client.AudioQuery(context.Background(), "テスト", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if query.SpeedScale != 1.2 {
		t.Errorf("SpeedScale: got %v, want 1.2", query.SpeedScale)
	}
	if query.PitchScale != 0.1 {
		t.Errorf("PitchScale: got %v, want 0.1", query.PitchScale)
	}
	if query.IntonationScale != 0.8 {
		t.Errorf("IntonationScale: got %v, want 0.8", query.IntonationScale)
	}
	if query.VolumeScale != 1.5 {
		t.Errorf("VolumeScale: got %v, want 1.5", query.VolumeScale)
	}
}

func TestHTTPVoicevoxClient_AudioQuery_ReturnsErrorOnNonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "engine error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.AudioQuery(context.Background(), "テスト", 3)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestHTTPVoicevoxClient_Synthesis_SendsCorrectRequest(t *testing.T) {
	var gotMethod, gotPath, gotSpeaker string
	var gotBody AudioQuery
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotSpeaker = r.URL.Query().Get("speaker")
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "audio/wav")
		w.Write([]byte("FAKEWAV"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	query := &AudioQuery{SpeedScale: 1.0, PitchScale: 0.0}
	_, err := client.Synthesis(context.Background(), query, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", gotMethod)
	}
	if gotPath != "/synthesis" {
		t.Errorf("path: got %s, want /synthesis", gotPath)
	}
	if gotSpeaker != "3" {
		t.Errorf("speaker: got %s, want 3", gotSpeaker)
	}
	if gotBody.SpeedScale != 1.0 {
		t.Errorf("body SpeedScale: got %v, want 1.0", gotBody.SpeedScale)
	}
}

func TestHTTPVoicevoxClient_Synthesis_ReturnsWAVBytes(t *testing.T) {
	expected := []byte("FAKEWAVDATA")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		w.Write(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	got, err := client.Synthesis(context.Background(), &AudioQuery{}, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(got) != string(expected) {
		t.Errorf("wav bytes: got %q, want %q", got, expected)
	}
}

func TestHTTPVoicevoxClient_Synthesis_ReturnsErrorOnNonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "engine error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Synthesis(context.Background(), &AudioQuery{}, 3)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
