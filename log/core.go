package log

import (
	"github.com/hinha/floody/log/diode"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"time"
)

const (
	//intervalWrite   = time.Duration(5) * time.Minute
	bufferSize      = 1024 * 1024
	bufferSizeDebug = 1024
)

func newWriter(lj *lumberjack.Logger, interval time.Duration, Alerter diode.AlertFunc) (io.Writer, *lumberjack.Logger) {
	d := diode.NewWriter(lj, bufferSize, interval, Alerter)
	return d, lj
}

func getStdout(interval time.Duration, Alerter diode.AlertFunc) io.Writer {
	w := diode.NewWriter(os.Stdout, bufferSize, interval, Alerter)
	return w
}

// jsonEncoder returns a zapcore.Core that encodes log messages as JSON.
func jsonEncoder(w io.Writer, debug bool, cfg zapcore.EncoderConfig, lvl zap.AtomicLevel) zapcore.Core {
	if debug {
		return zapcore.NewCore(zapcore.NewJSONEncoder(cfg), &zapcore.BufferedWriteSyncer{
			WS:   zapcore.AddSync(w),
			Size: bufferSizeDebug,
		}, lvl)
	}
	return zapcore.NewCore(zapcore.NewJSONEncoder(cfg), &zapcore.BufferedWriteSyncer{
		WS:   zapcore.AddSync(w),
		Size: bufferSize,
	}, lvl)
}

func consoleEncoder(w io.Writer, cfg zapcore.EncoderConfig, lvl zap.AtomicLevel) zapcore.Core {
	return zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(w),
		lvl,
	)
}
