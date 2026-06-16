package testutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/testutil"
)

func TestWriteTempFile_ReturnsPathToWrittenFile(t *testing.T) {
	content := []byte("hello: world\n")
	path := testutil.WriteTempFile(t, "test.yaml", content)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
	if filepath.Base(path) != "test.yaml" {
		t.Errorf("filename = %q, want %q", filepath.Base(path), "test.yaml")
	}
}

func TestWriteTempFile_FileIsCleanedUpAfterTest(t *testing.T) {
	var savedPath string
	t.Run("inner", func(t *testing.T) {
		savedPath = testutil.WriteTempFile(t, "cleanup.yaml", []byte("data"))
	})
	if _, err := os.Stat(savedPath); !os.IsNotExist(err) {
		t.Errorf("expected file to be cleaned up, but it still exists: %s", savedPath)
	}
}

func TestStrPtr(t *testing.T) {
	tests := []struct {
		name string
		v    string
	}{
		{"empty string", ""},
		{"non-empty string", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.StrPtr(tt.v)
			if got == nil {
				t.Fatal("StrPtr returned nil")
			}
			if *got != tt.v {
				t.Errorf("*StrPtr(%q) = %q, want %q", tt.v, *got, tt.v)
			}
		})
	}
}

func TestFloat64Ptr(t *testing.T) {
	tests := []struct {
		name string
		v    float64
	}{
		{"zero", 0.0},
		{"positive", 1.5},
		{"negative", -0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.Float64Ptr(tt.v)
			if got == nil {
				t.Fatal("Float64Ptr returned nil")
			}
			if *got != tt.v {
				t.Errorf("*Float64Ptr(%v) = %v, want %v", tt.v, *got, tt.v)
			}
		})
	}
}

func TestBoolPtr(t *testing.T) {
	tests := []struct {
		name string
		v    bool
	}{
		{"false", false},
		{"true", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.BoolPtr(tt.v)
			if got == nil {
				t.Fatal("BoolPtr returned nil")
			}
			if *got != tt.v {
				t.Errorf("*BoolPtr(%v) = %v, want %v", tt.v, *got, tt.v)
			}
		})
	}
}
