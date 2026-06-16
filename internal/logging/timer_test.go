package logging_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/logging"
)

func splitLogLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func TestStartStep_LogsStartMessage(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(logging.NewTextHandler(&buf, slog.LevelInfo))

	done := logging.StartStep(logger, "開始")
	defer done("")

	lines := splitLogLines(buf.String())
	if len(lines) < 1 || !strings.Contains(lines[0], "開始") {
		t.Errorf("start message not found in %q", buf.String())
	}
}

func TestStartStep_LogsDoneMessageWithElapsed(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(logging.NewTextHandler(&buf, slog.LevelInfo))

	done := logging.StartStep(logger, "開始")
	done("")

	lines := splitLogLines(buf.String())
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), buf.String())
	}
	if !strings.Contains(lines[1], "完了") {
		t.Errorf("done message not found in line: %q", lines[1])
	}
	if !strings.Contains(lines[1], "s)") {
		t.Errorf("elapsed time not found in done line: %q", lines[1])
	}
}

func TestStartStep_EmptySummary_DoneFormatIsElapsedOnly(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(logging.NewTextHandler(&buf, slog.LevelInfo))

	done := logging.StartStep(logger, "開始")
	done("")

	lines := splitLogLines(buf.String())
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), buf.String())
	}
	doneLine := lines[1]
	// "完了 (T.Ts)" format: no extra comma before elapsed
	if !strings.Contains(doneLine, "完了 (") {
		t.Errorf("done line should contain '完了 (': %q", doneLine)
	}
	// should not have a comma (no summary prefix)
	if strings.Contains(doneLine, ", ") {
		t.Errorf("done line should not have summary separator when summary is empty: %q", doneLine)
	}
}

func TestStartStep_WithSummary_DoneFormatIncludesSummary(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(logging.NewTextHandler(&buf, slog.LevelInfo))

	done := logging.StartStep(logger, "開始")
	done("5クリップ")

	lines := splitLogLines(buf.String())
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), buf.String())
	}
	doneLine := lines[1]
	if !strings.Contains(doneLine, "完了 (5クリップ,") {
		t.Errorf("done line should contain summary: %q", doneLine)
	}
	if !strings.Contains(doneLine, "s)") {
		t.Errorf("done line should contain elapsed: %q", doneLine)
	}
}

func TestStartStep_AttrsAppearedInBothStartAndDone(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(logging.NewTextHandler(&buf, slog.LevelInfo))

	done := logging.StartStep(logger, "開始", slog.String("corner", "テストコーナー"))
	done("")

	lines := splitLogLines(buf.String())
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], "corner=テストコーナー") {
		t.Errorf("start line should contain attr: %q", lines[0])
	}
	if !strings.Contains(lines[1], "corner=テストコーナー") {
		t.Errorf("done line should contain attr: %q", lines[1])
	}
}

func TestStartStep_MultipleAttrs(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(logging.NewTextHandler(&buf, slog.LevelInfo))

	done := logging.StartStep(logger, "開始",
		slog.String("corner", "コーナーA"),
		slog.Int("count", 3),
	)
	done("3記事")

	output := buf.String()
	if !strings.Contains(output, "corner=コーナーA") {
		t.Errorf("output should contain corner attr: %q", output)
	}
	if !strings.Contains(output, "count=3") {
		t.Errorf("output should contain count attr: %q", output)
	}
}
