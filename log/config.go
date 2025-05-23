package log

import (
	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"time"
)

type Config struct {
	Base zap.Config

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int
	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int
	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool

	// Filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.  It uses <processname>-lumberjack.log in
	// os.TempDir() if empty.
	Filename string

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int

	Interval time.Duration
}

func baseZapProductionConfig() zap.Config {
	return zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:      "json",
		EncoderConfig: zap.NewProductionEncoderConfig(),
	}
}

func baseZapDevelopmentConfig() zap.Config {
	return zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

// NewProductionConfig is a reasonable production logging configuration.
// Logging is enabled at InfoLevel and above.
//
// It uses a JSON encoder, writes to standard error, and enables sampling.
// Stack traces are automatically included on logs of ErrorLevel and above.
func NewProductionConfig() Config {
	return Config{
		Base:       baseZapProductionConfig(),
		MaxSize:    100, // 100MB
		MaxBackups: 3,
		LocalTime:  true,
		Compress:   true,
		Filename:   "app.log", // TODO Use option to set filename
		MaxAge:     30,
		Interval:   time.Duration(15) * time.Microsecond,
	}
}

func NewDevelopmentConfig() Config {
	return Config{
		Base:       baseZapDevelopmentConfig(),
		MaxSize:    10, // 10MB
		MaxBackups: 1,
		LocalTime:  true,
		Compress:   false,
		Filename:   "app.log", // TODO Use option to set filename
		MaxAge:     30,
		Interval:   time.Microsecond,
	}
}

func writerJack(c Config) (io.Writer, *lumberjack.Logger) {
	lg := &lumberjack.Logger{
		Filename:   c.Filename,
		MaxSize:    c.MaxSize,
		MaxAge:     c.MaxAge,
		MaxBackups: c.MaxBackups,
		LocalTime:  c.LocalTime,
	}
	return newWriter(lg, c.Interval, nil)
}
