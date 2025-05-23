package log

import (
	"github.com/hinha/floody/log/diode"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var DefaultLogger = NewLogger(NewDevelopmentConfig()).WithOptions(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

func init() {
	defer DefaultLogger.Sync()
}

type LoggerI interface {
	Sync() error
	Core() zapcore.Core
	WithOptions(opts ...zap.Option) *zap.Logger
	GetLogger() *Logger
}

type Logger struct {
	logger  *zap.Logger
	config  Config
	core    zapcore.Core
	cores   []zapcore.Core
	rotate  *lumberjack.Logger
	optsZap []zap.Option
	alerter diode.AlertFunc
}

func NewLogger(config Config, opts ...Option) LoggerI {
	log := &Logger{}
	for _, opt := range opts {
		opt.apply(log)
	}

	core := make([]zapcore.Core, 0)
	core = append(core, log.cores...)
	cslEncoder := consoleEncoder(getStdout(config.Interval, log.alerter), config.Base.EncoderConfig, config.Base.Level)
	logfile, lb := writerJack(config)
	if config.Base.Encoding == "all" {
		log.rotate = lb
		fileEncoder := jsonEncoder(logfile, config.Base.Development, config.Base.EncoderConfig, config.Base.Level)
		core = append(core, fileEncoder, cslEncoder)
	} else if config.Base.Encoding == "json" {
		log.rotate = lb
		fileEncoder := jsonEncoder(logfile, config.Base.Development, config.Base.EncoderConfig, config.Base.Level)
		core = append(core, fileEncoder)
	} else {
		core = append(core, cslEncoder)
	}

	cr := zapcore.NewTee(core...)
	log.core = cr
	log.logger = zap.New(log.core, log.optsZap...)
	return log
}

func (log *Logger) GetLogger() *Logger {
	return log
}

// Sync wrap sync
func (log *Logger) Sync() error {
	err := log.core.Sync()
	if err != nil {
		return err
	}

	if log.rotate != nil {
		return log.rotate.Rotate()
	}

	return log.core.Sync()
}

func (log *Logger) Core() zapcore.Core {
	return log.core
}

// WithOptions clones the current Logger, applies the supplied Options, and
// returns the resulting Logger. It's safe to use concurrently.
func (log *Logger) WithOptions(opts ...zap.Option) *zap.Logger {
	return log.logger.WithOptions(opts...)
}
