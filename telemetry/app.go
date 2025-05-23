package telemetry

import (
	"context"
	"github.com/shirou/gopsutil/v3/cpu"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"time"
)

type Application[M metric.Meter, T trace.Tracer] struct {
	ctx          context.Context
	tracer       T
	traceOptions []trace.TracerOption

	meter        M
	meterOptions []metric.MeterOption
	spanOptions  []trace.SpanStartOption
}

func (app *Application[M, T]) GetMeter() M {
	return app.meter
}

func (app *Application[M, T]) observeMeter(provider metric.MeterProvider) error {
	err := otelruntime.Start(otelruntime.WithMeterProvider(provider), otelruntime.WithMinimumReadMemStatsInterval(time.Second*15))
	if err != nil {
		return err
	}

	cpuGauge, err := app.meter.Float64ObservableGauge("go.cpu.usage.percent", metric.WithDescription("Current system CPU usage percentage"))
	if err != nil {
		return err
	}

	_, err = app.meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		percentages, err := cpu.Percent(0, false)
		if err != nil {
			return err
		}
		if len(percentages) > 0 {
			observer.ObserveFloat64(cpuGauge, percentages[0])
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (app *Application[M, T]) GetTracer() T {
	return app.tracer
}

// AppOption applies an option to a factory.
type AppOption[M metric.Meter, T trace.Tracer] interface {
	apply(*Application[metric.Meter, trace.Tracer]) *Application[metric.Meter, trace.Tracer]
}

type appOptionFunc func(*Application[metric.Meter, trace.Tracer]) *Application[metric.Meter, trace.Tracer]

func (fn appOptionFunc) apply(cfg *Application[metric.Meter, trace.Tracer]) *Application[metric.Meter, trace.Tracer] {
	return fn(cfg)
}

func WithSpanOption(options ...trace.SpanStartOption) AppOption[metric.Meter, trace.Tracer] {
	return appOptionFunc(func(cfg *Application[metric.Meter, trace.Tracer]) *Application[metric.Meter, trace.Tracer] {
		if cfg.tracer != nil && cfg.meter != nil {
			cfg.spanOptions = append(cfg.spanOptions, options...)
		}
		return cfg
	})
}
