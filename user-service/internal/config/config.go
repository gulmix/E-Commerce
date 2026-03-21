package config

type Config struct {
	GRPCPort           string `mapstructure:"GRPC_PORT"`
	DBDsn              string `mapstructure:"DB_DSN"`
	LogLevel           string `mapstructure:"LOG_LEVEL"`
	JWTSecret          string `mapstructure:"JWT_SECRET"`
	AccessExpiryMin    int    `mapstructure:"JWT_ACCESS_EXPIRY_MIN"`
	RefreshExpiryHours int    `mapstructure:"JWT_REFRESH_EXPIRY_HOURS"`
	MetricsPort        string `mapstructure:"METRICS_PORT"`
	OTELEndpoint       string `mapstructure:"OTEL_ENDPOINT"`
}
