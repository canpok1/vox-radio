package logging_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/logging"
)

func splitLogLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func newTestLogger(t *testing.T) (*slog.Logger, *strings.Builder) {
	t.Helper()
	var buf strings.Builder
	return slog.New(logging.NewTextHandler(&buf, slog.LevelInfo)), &buf
}

func TestStartStep_LogsStartMessage(t *testing.T) {
	logger, buf := newTestLogger(t)

	done := logging.StartStep(context.Background(), logger, "開始")
	defer done("")

	lines := splitLogLines(buf.String())
	if len(lines) < 1 || !strings.Contains(lines[0], "開始") {
		t.Errorf("start message not found in %q", buf.String())
	}
}

func TestStartStep_EmptySummary(t *testing.T) {
	logger, buf := newTestLogger(t)

	done := logging.StartStep(context.Background(), logger, "開始")
	done("")

	lines := splitLogLines(buf.String())
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), buf.String())
	}
	doneLine := lines[1]
	if !strings.Contains(doneLine, "完了") {
		t.Errorf("done message not found in line: %q", doneLine)
	}
	if !strings.Contains(doneLine, "s)") {
		t.Errorf("elapsed time not found in done line: %q", doneLine)
	}
	// "完了 (T.Ts)" format: no comma (no summary prefix)
	if strings.Contains(doneLine, ", ") {
		t.Errorf("done line should not have summary separator when summary is empty: %q", doneLine)
	}
}

func TestStartStep_WithSummary_DoneFormatIncludesSummary(t *testing.T) {
	logger, buf := newTestLogger(t)

	done := logging.StartStep(context.Background(), logger, "開始")
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
	cases := []struct {
		name  string
		attrs []slog.Attr
		wants []string
	}{
		{
			name:  "single attr",
			attrs: []slog.Attr{slog.String("corner", "テストコーナー")},
			wants: []string{"corner=テストコーナー"},
		},
		{
			name:  "multiple attrs",
			attrs: []slog.Attr{slog.String("corner", "コーナーA"), slog.Int("count", 3)},
			wants: []string{"corner=コーナーA", "count=3"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, buf := newTestLogger(t)

			done := logging.StartStep(context.Background(), logger, "開始", tc.attrs...)
			done("")

			lines := splitLogLines(buf.String())
			if len(lines) < 2 {
				t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), buf.String())
			}
			for _, want := range tc.wants {
				if !strings.Contains(lines[0], want) {
					t.Errorf("start line should contain %q: %q", want, lines[0])
				}
				if !strings.Contains(lines[1], want) {
					t.Errorf("done line should contain %q: %q", want, lines[1])
				}
			}
		})
	}
}
