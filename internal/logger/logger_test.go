package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitialize(t *testing.T) {
	for _, level := range []zapcore.Level{
		zap.DebugLevel,
		zap.InfoLevel,
		zap.WarnLevel,
		zap.ErrorLevel,
		zap.FatalLevel,
	} {
		atomicLevel := zap.NewAtomicLevelAt(level)
		err := Initialize(atomicLevel.String())
		require.NoError(t, err)
		assert.Equal(t, level, Log.Level())
	}
	// clear logger so that other tests do not sent spam messages
	Log = zap.NewNop()
}
