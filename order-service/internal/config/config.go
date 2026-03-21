package config

type Config struct {
	GRPCPort           string `mapstructure:"GRPC_PORT"`
	DBDsn              string `mapstructure:"DB_DSN"`
	LogLevel           string `mapstructure:"LOG_LEVEL"`
	UserServiceAddr    string `mapstructure:"USER_SERVICE_ADDR"`
	ProductServiceAddr string `mapstructure:"PRODUCT_SERVICE_ADDR"`
	RabbitMQURL        string `mapstructure:"RABBITMQ_URL"`
	MetricsPort        string `mapstructure:"METRICS_PORT"`
	OTELEndpoint       string `mapstructure:"OTEL_ENDPOINT"`
}
