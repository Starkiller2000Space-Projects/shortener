// Package logger initializes and provides a global zap logger.
package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// Log is the global logger instance. It is initialized with a no-op logger by default.
// Use Initialize to set up a production logger with the desired level.
var Log *zap.Logger = zap.NewNop()

// Initialize configures the global logger with the given log level.
// It uses zap's production configuration and sets the level.
// Returns an error if the level string is invalid or the logger fails to build.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	Log = zl
	return nil
}
