package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config agrega toda a configuração da aplicação.
// Cada subseção pode ser usada de forma independente pelos módulos.
type Config struct {
	Env  string     `mapstructure:"env"`
	HTTP HTTPConfig `mapstructure:"http"`
	Log  LogConfig  `mapstructure:"log"`
}

// HTTPConfig carrega portas por módulo e timeouts comuns.
type HTTPConfig struct {
	ReadPort        int           `mapstructure:"read_port"`
	ExtractPort     int           `mapstructure:"extract_port"`
	ObservePort     int           `mapstructure:"observe_port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

// Load lê variáveis de ambiente prefixadas com RHAXIS_ e devolve a Config.
// Ex.: RHAXIS_HTTP_READ_PORT=9090  -> Config.HTTP.ReadPort = 9090
func Load() (*Config, error) {
	v := viper.New()

	v.SetEnvPrefix("RHAXIS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// defaults
	v.SetDefault("env", "dev")
	v.SetDefault("log.level", "info")
	v.SetDefault("http.read_port", 8081)
	v.SetDefault("http.extract_port", 8082)
	v.SetDefault("http.observe_port", 8083)
	v.SetDefault("http.read_timeout", "10s")
	v.SetDefault("http.write_timeout", "10s")
	v.SetDefault("http.idle_timeout", "60s")
	v.SetDefault("http.shutdown_timeout", "15s")

	// Viper precisa "ver" cada chave para que AutomaticEnv funcione com Unmarshal.
	// As chaves abaixo cobrem todos os campos do struct.
	for _, key := range []string{
		"env", "log.level",
		"http.read_port", "http.extract_port", "http.observe_port",
		"http.read_timeout", "http.write_timeout",
		"http.idle_timeout", "http.shutdown_timeout",
	} {
		_ = v.BindEnv(key)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}
	return &cfg, nil
}
