package log

import (
	"github.com/hinha/floody/log/diode"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

// An Option configures a Logger.
type Option interface {
	apply(*Logger)
}

// optionFunc wraps a func. so it satisfies the Option interface.
type optionFunc func(*Logger)

func (f optionFunc) apply(log *Logger) {
	f(log)
}

func WithAlerter(alerter diode.AlertFunc) Option {
	return optionFunc(func(log *Logger) {
		log.alerter = alerter
	})
}

func WithCore(core ...zapcore.Core) Option {
	return optionFunc(func(log *Logger) {
		log.cores = append(log.cores, core...)
	})
}

func WithZapOptions(opts ...zap.Option) Option {
	return optionFunc(func(log *Logger) {
		log.optsZap = append(log.optsZap, opts...)
	})
}

func WithLogFileName(name string) Option {
	return optionFunc(func(log *Logger) {
		log.config.Filename = name
	})
}

func WithLogMaxSize(size int) Option {
	return optionFunc(func(log *Logger) {
		log.config.MaxSize = size
	})
}

func WithLogMaxBackups(backups int) Option {
	return optionFunc(func(log *Logger) {
		log.config.MaxBackups = backups
	})
}

func WithLogLocalTime(local bool) Option {
	return optionFunc(func(log *Logger) {
		log.config.LocalTime = local
	})
}

func WithLogCompress(compress bool) Option {
	return optionFunc(func(log *Logger) {
		log.config.Compress = compress
	})
}

func WithLogMaxAge(age int) Option {
	return optionFunc(func(log *Logger) {
		log.config.MaxAge = age
	})
}

func WithLogInterval(interval time.Duration) Option {
	return optionFunc(func(log *Logger) {
		log.config.Interval = interval
	})
}
