package config

import (
	"fmt"
	"log/slog"
	"strings"
)

type LogLevel slog.Level

func (lld *LogLevel) Decode(value string) error {
	var loggerLevels = map[string]slog.Level{
		"error": slog.LevelError,
		"warn":  slog.LevelWarn,
		"info":  slog.LevelInfo,
		"debug": slog.LevelDebug,
	}
	// Reject unknown values instead of silently defaulting to debug (0), which
	// would leak verbose logs in production on a typo.
	lvl, ok := loggerLevels[strings.ToLower(strings.TrimSpace(value))]
	if !ok {
		return fmt.Errorf("invalid log level %q: want one of error, warn, info, debug", value)
	}
	*lld = LogLevel(lvl)
	return nil
}
