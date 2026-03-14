package config

import (
	"strings"

	"github.com/spf13/viper"
)

func Load(out any) error {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	_ = v.ReadInConfig()

	return v.Unmarshal(out)
}
