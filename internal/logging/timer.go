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
// ctx is forwarded to slog so context-propagated values (e.g. trace IDs) appear in both lines.
func StartStep(ctx context.Context, logger *slog.Logger, startMsg string, attrs ...slog.Attr) func(summary string) {
	start := time.Now()
	logger.LogAttrs(ctx, slog.LevelInfo, startMsg, attrs...)
	return func(summary string) {
		elapsed := time.Since(start).Seconds()
		var msg string
		if summary == "" {
			msg = fmt.Sprintf("完了 (%.1fs)", elapsed)
		} else {
			msg = fmt.Sprintf("完了 (%s, %.1fs)", summary, elapsed)
		}
		logger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
	}
}
