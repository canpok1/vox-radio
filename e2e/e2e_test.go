//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
)

// binaryPath は TestMain でビルドした vox-radio バイナリの絶対パス。
var binaryPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "vox-radio-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}

	binaryPath = filepath.Join(tmpDir, "vox-radio")
	build := exec.Command("go", "build", "-o", binaryPath, "github.com/canpok1/vox-radio/cmd/vox-radio")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build vox-radio binary: %v\n", err)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestFeatures(t *testing.T) {
	tags := ""
	if !hasFFmpeg() {
		t.Log("ffmpeg/ffprobe が見つからないため @ffmpeg タグ付きシナリオをスキップします")
		tags = "~@ffmpeg"
	}

	suite := godog.TestSuite{
		Name:                "vox-radio",
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
			Strict:   true,
			Tags:     tags,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("e2e シナリオが失敗しました")
	}
}

func hasFFmpeg() bool {
	_, errFFmpeg := exec.LookPath("ffmpeg")
	_, errFFprobe := exec.LookPath("ffprobe")
	return errFFmpeg == nil && errFFprobe == nil
}
