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

func TestPtr_ReturnsPointerToValue(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		cases := []string{"", "hello"}
		for _, v := range cases {
			got := testutil.Ptr(v)
			if *got != v {
				t.Errorf("*Ptr(%q) = %q, want %q", v, *got, v)
			}
		}
	})
	t.Run("float64", func(t *testing.T) {
		cases := []float64{0.0, 1.5, -0.5}
		for _, v := range cases {
			got := testutil.Ptr(v)
			if *got != v {
				t.Errorf("*Ptr(%v) = %v, want %v", v, *got, v)
			}
		}
	})
	t.Run("bool", func(t *testing.T) {
		cases := []bool{false, true}
		for _, v := range cases {
			got := testutil.Ptr(v)
			if *got != v {
				t.Errorf("*Ptr(%v) = %v, want %v", v, *got, v)
			}
		}
	})
}
