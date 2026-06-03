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
	for _, sub := range []string{"init", "episodegen", "config", "feedgen"} {
		if !strings.Contains(out, sub) {
			t.Errorf("root help missing subcommand %q", sub)
		}
	}
	// コマンドの行頭パターン（"  <name>  "）で検索することで、説明文中の単語と区別する
	for _, sub := range []string{"collect", "rundown", "script", "synth", "assemble", "manifest", "run", "publish", "prune"} {
		if strings.Contains(out, "\n  "+sub+" ") {
			t.Errorf("root help should not list %q as a top-level subcommand", sub)
		}
	}
}

func TestCollectMissingOut(t *testing.T) {
	cmd := cli.NewRootCmd()
	errBuf := &bytes.Buffer{}
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"episodegen", "collect"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out is missing")
	}
}

func TestSynthMissingIn(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "synth", "--out-dir", "/tmp"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --in is missing")
	}
}

func TestSynthMissingOutDir(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "synth", "--in", "/tmp/script.json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out-dir is missing")
	}
}

func TestAssembleMissingIn(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "assemble", "--clips", "/tmp", "--out", "/tmp/ep.mp3"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --in is missing")
	}
}

func TestAssembleMissingClips(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "assemble", "--in", "/tmp/script.json", "--out", "/tmp/ep.mp3"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --clips is missing")
	}
}

func TestAssembleMissingOut(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "assemble", "--in", "/tmp/script.json", "--clips", "/tmp"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out is missing")
	}
}

func TestScriptMissingOut(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"episodegen", "script"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --out is missing")
	}
}

func TestSpecRequired(t *testing.T) {
	// --spec はデフォルト値を持たず、各サブコマンドで必須であること。
	// assemble は assets を任意で読み込むため --spec は optional（意図的に対象外）。
	tests := []struct {
		name string
		args []string
	}{
		{name: "episodegen", args: []string{"episodegen"}},
		{name: "collect", args: []string{"episodegen", "collect"}},
		{name: "rundown", args: []string{"episodegen", "rundown"}},
		{name: "script", args: []string{"episodegen", "script"}},
		{name: "manifest", args: []string{"episodegen", "manifest"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cli.NewRootCmd()
			errBuf := &bytes.Buffer{}
			cmd.SetErr(errBuf)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error when --spec is missing for %q", tt.name)
			}
			if !strings.Contains(err.Error(), "spec") || !strings.Contains(err.Error(), "not set") {
				t.Errorf("%s: error should report required --spec flag, got: %v", tt.name, err)
			}
		})
	}
}

func TestRootCmdDisableAutoGenTag(t *testing.T) {
	cmd := cli.NewRootCmd()
	if !cmd.DisableAutoGenTag {
		t.Error("DisableAutoGenTag must be true to keep make docs idempotent")
	}
}

func TestRootVersion(t *testing.T) {
	cmd := cli.NewRootCmd()
	if cmd.Version != "dev" {
		t.Errorf("expected version %q, got %q", "dev", cmd.Version)
	}
}

func TestRootVersionFlag(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "vox-radio version dev") {
		t.Errorf("--version output should contain %q, got %q", "vox-radio version dev", out)
	}
}

func TestSubcommandHelp(t *testing.T) {
	// トップレベルのサブコマンド
	for _, sub := range []string{"init", "config"} {
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

	// episodegen 本体
	t.Run("episodegen", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"episodegen", "--help"})
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "--") {
			t.Errorf("episodegen help should contain flag descriptions")
		}
	})

	// episodegen 配下のサブコマンド
	for _, sub := range []string{"collect", "rundown", "synth", "assemble", "manifest", "script"} {
		t.Run("episodegen/"+sub, func(t *testing.T) {
			cmd := cli.NewRootCmd()
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetArgs([]string{"episodegen", sub, "--help"})
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			out := buf.String()
			if !strings.Contains(out, "--") {
				t.Errorf("episodegen %s help should contain flag descriptions", sub)
			}
		})
	}

	// feedgen 本体
	t.Run("feedgen", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"feedgen", "--help"})
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "--") {
			t.Errorf("feedgen help should contain flag descriptions")
		}
		// check サブコマンドが列挙されること
		if !strings.Contains(out, "\n  check ") {
			t.Errorf("feedgen help should list check subcommand")
		}
	})

	// feedgen check サブコマンド
	t.Run("feedgen/check", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"feedgen", "check", "--help"})
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
