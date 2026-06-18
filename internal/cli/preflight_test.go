package cli

import (
	"errors"
	"strings"
	"testing"
)

var testReadmeURL = referenceURL("README.md")

func setLookPath(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	orig := lookPath
	lookPath = fn
	t.Cleanup(func() { lookPath = orig })
}

func TestRequireMediaTools_AllPresent(t *testing.T) {
	setLookPath(t, func(_ string) (string, error) { return "/usr/bin/tool", nil })

	if err := requireMediaTools(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRequireMediaTools_OneMissing(t *testing.T) {
	setLookPath(t, func(name string) (string, error) {
		if name == "ffprobe" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + name, nil
	})

	err := requireMediaTools()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ffprobe") {
		t.Errorf("want 'ffprobe' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), testReadmeURL) {
		t.Errorf("want README URL %q in error, got: %v", testReadmeURL, err)
	}
}

func TestRequireMediaTools_BothMissing(t *testing.T) {
	setLookPath(t, func(_ string) (string, error) { return "", errors.New("not found") })

	err := requireMediaTools()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ffmpeg") {
		t.Errorf("want 'ffmpeg' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "ffprobe") {
		t.Errorf("want 'ffprobe' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), testReadmeURL) {
		t.Errorf("want README URL %q in error, got: %v", testReadmeURL, err)
	}
}
