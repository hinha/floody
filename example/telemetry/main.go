package main

import (
	"github.com/hinha/floody/telemetry"
)

var options = &telemetry.Options{
	AppName: "my-example-server-test",
	EndpointMeter: telemetry.Endpoint{
		Endpoint: "otlpmetric.hinha.web.id",
		WithHTTP: true,
	},
	Meter:    builder.Meter{},
	Insecure: true,
	EndpointTrace: telemetry.Endpoint{
		Endpoint: "otlptrace.hinha.web.id:443",
		WithHTTP: false,
		Debug:    true,
	},
}

func main() {
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	//defer cancel()

	// Init Telemetry
	otl := telemetry.NewFactory(options)
	//closers, err := otl.MeterOptions().
	//	TracerOptions().
	//	Build(ctx)
	//if err != nil {
	//	otl.GetLogger().Fatal("Telemetry error", zap.Error(err))
	//}
	//defer func() {
	//	for _, closer := range closers {
	//		closer(ctx)
	//	}
	//}()
	//
	//middleware := telemetry.NewMiddlewareWithConfig(otl)
	//mux := http.NewServeMux()
	//
	//mux.Handle("/", middleware.Handler("index", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//	// Seed dengan waktu saat ini
	//	rand.Seed(time.Now().UnixNano())
	//	randomInt := rand.Intn(200)
	//
	//	time.Sleep(time.Duration(randomInt) * time.Millisecond) // simulate work
	//	w.Write([]byte(fmt.Sprintf("Hello from OpenTelemetry! duration: %d", randomInt)))
	//})))
	//
	//log.Print("Listening on :8081...")
	//err = http.ListenAndServe(":8081", mux)
	//if err != nil {
	//	log.Fatal(err)
	//}
}
