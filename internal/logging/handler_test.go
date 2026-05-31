package logging_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/canpok1/vox-radio/internal/logging"
)

var fixedTime = time.Date(2026, 5, 31, 12, 1, 3, 0, time.UTC)

func record(level slog.Level, msg string, attrs ...slog.Attr) slog.Record {
	r := slog.NewRecord(fixedTime, level, msg, 0)
	for _, a := range attrs {
		r.AddAttrs(a)
	}
	return r
}

func TestTextHandler_FormatsWithStep(t *testing.T) {
	var buf strings.Builder
	h := logging.NewTextHandler(&buf, slog.LevelInfo)
	hl := h.WithAttrs([]slog.Attr{slog.String("step", "collect")})

	if err := hl.Handle(context.Background(), record(slog.LevelInfo, "開始")); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	got := buf.String()
	if !strings.HasPrefix(got, "[12:01:03] collect: 開始") {
		t.Errorf("unexpected format: %q", got)
	}
}

func TestTextHandler_FormatsWithoutStep(t *testing.T) {
	var buf strings.Builder
	h := logging.NewTextHandler(&buf, slog.LevelInfo)

	if err := h.Handle(context.Background(), record(slog.LevelInfo, "メッセージ")); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	got := buf.String()
	if !strings.HasPrefix(got, "[12:01:03] メッセージ") {
		t.Errorf("unexpected format: %q", got)
	}
	if strings.Contains(got, "step") {
		t.Errorf("step should not appear in output: %q", got)
	}
}

func TestTextHandler_AppendsOtherAttrs(t *testing.T) {
	var buf strings.Builder
	h := logging.NewTextHandler(&buf, slog.LevelInfo)
	hl := h.WithAttrs([]slog.Attr{slog.String("step", "synth")})

	r := record(slog.LevelInfo, "完了", slog.Int("clips", 8))
	if err := hl.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "[12:01:03] synth: 完了") {
		t.Errorf("missing prefix: %q", got)
	}
	if !strings.Contains(got, "clips=8") {
		t.Errorf("missing attr: %q", got)
	}
}

func TestTextHandler_FiltersByLevel(t *testing.T) {
	var buf strings.Builder
	h := logging.NewTextHandler(&buf, slog.LevelInfo)

	if err := h.Handle(context.Background(), record(slog.LevelDebug, "デバッグ")); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("DEBUG should be filtered at INFO level, got: %q", buf.String())
	}
}

func TestTextHandler_Enabled(t *testing.T) {
	h := logging.NewTextHandler(&strings.Builder{}, slog.LevelInfo)
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("INFO should be enabled")
	}
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("DEBUG should not be enabled at INFO level")
	}
}

func TestTextHandler_EndsWithNewline(t *testing.T) {
	var buf strings.Builder
	h := logging.NewTextHandler(&buf, slog.LevelInfo)

	if err := h.Handle(context.Background(), record(slog.LevelInfo, "テスト")); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	got := buf.String()
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("output should end with newline: %q", got)
	}
}

func TestFanOutHandler_RoutesToAllHandlers(t *testing.T) {
	var buf1, buf2 strings.Builder
	h1 := logging.NewTextHandler(&buf1, slog.LevelInfo)
	h2 := logging.NewTextHandler(&buf2, slog.LevelDebug)

	fan := logging.NewFanOutHandler(h1, h2)

	if err := fan.Handle(context.Background(), record(slog.LevelInfo, "共通ログ")); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	if !strings.Contains(buf1.String(), "共通ログ") {
		t.Errorf("h1 should receive record: %q", buf1.String())
	}
	if !strings.Contains(buf2.String(), "共通ログ") {
		t.Errorf("h2 should receive record: %q", buf2.String())
	}
}

func TestFanOutHandler_FiltersDebugFromInfoHandler(t *testing.T) {
	var buf1, buf2 strings.Builder
	h1 := logging.NewTextHandler(&buf1, slog.LevelInfo)
	h2 := logging.NewTextHandler(&buf2, slog.LevelDebug)

	fan := logging.NewFanOutHandler(h1, h2)

	if err := fan.Handle(context.Background(), record(slog.LevelDebug, "詳細ログ")); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	if buf1.Len() != 0 {
		t.Errorf("h1 (INFO) should not receive DEBUG: %q", buf1.String())
	}
	if !strings.Contains(buf2.String(), "詳細ログ") {
		t.Errorf("h2 (DEBUG) should receive record: %q", buf2.String())
	}
}

func TestFanOutHandler_Enabled(t *testing.T) {
	h1 := logging.NewTextHandler(&strings.Builder{}, slog.LevelInfo)
	h2 := logging.NewTextHandler(&strings.Builder{}, slog.LevelDebug)

	fan := logging.NewFanOutHandler(h1, h2)

	if !fan.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("FanOut should be enabled for DEBUG (h2 accepts it)")
	}
}

func TestFanOutHandler_WithAttrs_PropagatesStep(t *testing.T) {
	var buf strings.Builder
	h := logging.NewTextHandler(&buf, slog.LevelInfo)
	fan := logging.NewFanOutHandler(h)
	fanWithStep := fan.WithAttrs([]slog.Attr{slog.String("step", "collect")})

	if err := fanWithStep.Handle(context.Background(), record(slog.LevelInfo, "進捗")); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	if !strings.Contains(buf.String(), "collect: 進捗") {
		t.Errorf("step should be propagated: %q", buf.String())
	}
}
