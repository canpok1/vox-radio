package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
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

func TestRequireLLMKey_OpenAI(t *testing.T) {
	cfg := &config.Config{LLM: config.LLMConfig{
		Provider: "openai",
		OpenAI:   &config.OpenAIConfig{APIKeyEnv: "TEST_OPENAI_KEY"},
	}}

	t.Run("set", func(t *testing.T) {
		t.Setenv("TEST_OPENAI_KEY", "secret")
		if err := requireLLMKey(cfg); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("unset", func(t *testing.T) {
		t.Setenv("TEST_OPENAI_KEY", "")
		err := requireLLMKey(cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "TEST_OPENAI_KEY") {
			t.Errorf("want env var name in error, got: %v", err)
		}
	})
}

func TestRequireLLMKey_DifyChat(t *testing.T) {
	cfg := &config.Config{LLM: config.LLMConfig{
		Provider: config.ProviderDifyChat,
		DifyChat: &config.DifyChatConfig{APIKeyEnv: "TEST_DIFY_KEY"},
	}}

	t.Run("set", func(t *testing.T) {
		t.Setenv("TEST_DIFY_KEY", "secret")
		if err := requireLLMKey(cfg); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("unset", func(t *testing.T) {
		t.Setenv("TEST_DIFY_KEY", "")
		err := requireLLMKey(cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "TEST_DIFY_KEY") {
			t.Errorf("want env var name in error, got: %v", err)
		}
	})
}

func TestCheckResources_AllPass(t *testing.T) {
	if err := checkResources(
		func() error { return nil },
		func() error { return nil },
	); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckResources_AggregatesFailures(t *testing.T) {
	err := checkResources(
		func() error { return errors.New("resource A missing") },
		func() error { return nil },
		func() error { return errors.New("resource B missing") },
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resource A missing") {
		t.Errorf("want 'resource A missing' in aggregated error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "resource B missing") {
		t.Errorf("want 'resource B missing' in aggregated error, got: %v", err)
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
