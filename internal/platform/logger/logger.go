package logger

import (
	"log/slog"
	"os"
	"strings"
)

var redactKeys = map[string]struct{}{
	"password":      {},
	"authorization": {},
	"access_token":  {},
	"jwt_secret":    {},
	"token":         {},
	"secret":        {},
}

func New(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if _, ok := redactKeys[strings.ToLower(a.Key)]; ok {
				return slog.String(a.Key, "[redacted]")
			}
			return a
		},
	})
	return slog.New(handler)
}
