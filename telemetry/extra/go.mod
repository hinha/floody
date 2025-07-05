module github.com/hinha/floody/telemetry/extra

go 1.23.0

toolchain go1.23.8

replace github.com/hinha/floody => ../../

require (
	github.com/go-redis/redis/extra/rediscmd/v8 v8.11.5
	github.com/go-redis/redis/v8 v8.11.5
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.3.2
	go.opentelemetry.io/otel v1.36.0
	go.opentelemetry.io/otel/trace v1.36.0
	gorm.io/gorm v1.30.0
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.36.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)
