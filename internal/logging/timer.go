package logging

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// StartStep logs startMsg with optional attrs and returns a completion function.
// The completion function accepts a summary string and logs
// "完了 (SUMMARY, T.Ts)" when summary is non-empty, or "完了 (T.Ts)" when empty.
// The same attrs are included in both the start and completion log entries.
func StartStep(logger *slog.Logger, startMsg string, attrs ...slog.Attr) func(summary string) {
	start := time.Now()
	logger.LogAttrs(context.Background(), slog.LevelInfo, startMsg, attrs...)
	return func(summary string) {
		elapsed := time.Since(start).Seconds()
		var msg string
		if summary == "" {
			msg = fmt.Sprintf("完了 (%.1fs)", elapsed)
		} else {
			msg = fmt.Sprintf("完了 (%s, %.1fs)", summary, elapsed)
		}
		logger.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
	}
}
