package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/cli"
)

func TestRootHelp(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, sub := range []string{"collect", "script", "synth", "assemble", "publish", "prune"} {
		if !strings.Contains(out, sub) {
			t.Errorf("root help missing subcommand %q", sub)
		}
	}
}

func TestCollectMissingOut(t *testing.T) {
	cmd := cli.NewRootCmd()
	errBuf := &bytes.Buffer{}
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"collect"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out is missing")
	}
}

func TestSynthMissingIn(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"synth", "--out-dir", "/tmp"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --in is missing")
	}
}

func TestSynthMissingOutDir(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"synth", "--in", "/tmp/script.json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out-dir is missing")
	}
}

func TestAssembleMissingIn(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assemble", "--clips", "/tmp", "--out", "/tmp/ep.mp3"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --in is missing")
	}
}

func TestAssembleMissingClips(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assemble", "--in", "/tmp/script.json", "--out", "/tmp/ep.mp3"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --clips is missing")
	}
}

func TestAssembleMissingOut(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"assemble", "--in", "/tmp/script.json", "--clips", "/tmp"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out is missing")
	}
}

func TestPublishMissingIn(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"publish", "--out-dir", "/tmp"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --in is missing")
	}
}

func TestPublishMissingOutDir(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"publish", "--in", "/tmp/ep.mp3"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out-dir is missing")
	}
}

func TestPruneMissingOutDir(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"prune"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out-dir is missing")
	}
}

func TestScriptMissingOut(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"script"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out is missing")
	}
}

func TestSubcommandHelp(t *testing.T) {
	for _, sub := range []string{"collect", "synth", "assemble", "publish", "prune", "script"} {
		t.Run(sub, func(t *testing.T) {
			cmd := cli.NewRootCmd()
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetArgs([]string{sub, "--help"})
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			out := buf.String()
			if !strings.Contains(out, "--") {
				t.Errorf("%s help should contain flag descriptions", sub)
			}
		})
	}
}
