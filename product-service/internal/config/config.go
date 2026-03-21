package config

type Config struct {
	GRPCPort     string `mapstructure:"GRPC_PORT"`
	DBDsn        string `mapstructure:"DB_DSN"`
	LogLevel     string `mapstructure:"LOG_LEVEL"`
	MetricsPort  string `mapstructure:"METRICS_PORT"`
	OTELEndpoint string `mapstructure:"OTEL_ENDPOINT"`
}
