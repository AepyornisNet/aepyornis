package container

import (
	"log/slog"
	"os"
	"strings"

	"github.com/cat-dealer/go-rand/v2"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Config struct {
	model.Config `mapstructure:",squash"`
}

func NewConfig() (*Config, error) {
	c := &Config{}

	if err := c.Load(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) GetDatabaseConfig() *model.Config {
	return &c.Config
}

func (c *Config) Load() error {
	viper.SetConfigName("workout-tracker")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("WT")

	viper.SetDefault("host", "")
	viper.SetDefault("bind", "[::]:8080")
	viper.SetDefault("web_root", "")
	viper.SetDefault("logging", true)
	viper.SetDefault("debug", false)
	viper.SetDefault("offline", false)
	viper.SetDefault("database_driver", "sqlite")
	viper.SetDefault("dsn", "./database.db")
	viper.SetDefault("registration_disabled", false)
	viper.SetDefault("socials_disabled", false)
	viper.SetDefault("worker_delay_seconds", 60)
	viper.SetDefault("activity_pub_active", false)

	for _, envVar := range []string{
		"host",
		"bind",
		"web_root",
		"jwt_encryption_key",
		"jwt_encryption_key_file",
		"logging",
		"offline",
		"debug",
		"database_driver",
		"dsn",
		"dsn_file",
		"registration_disabled",
		"socials_disabled",
		"worker_delay_seconds",
		"activity_pub_active",
	} {
		if err := viper.BindEnv(envVar); err != nil {
			return err
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return viper.Unmarshal(c)
}

func (c *Config) Reset(db *gorm.DB) error {
	if err := c.Load(); err != nil {
		return err
	}

	return c.UpdateFromDatabase(db)
}

func (c *Config) SetDSN(logger *slog.Logger) {
	if c.DSN != "" || c.DSNFile == "" {
		return
	}

	if logger != nil {
		logger.Info("reading DSNFile", "file", c.DSNFile)
	}

	dsn, err := os.ReadFile(c.DSNFile)
	if err != nil {
		if logger != nil {
			logger.Error("could not read DSN file", "error", err)
		}
		return
	}

	c.DSN = strings.TrimSpace(string(dsn))
}

func (c *Config) JWTSecret() []byte {
	if c.JWTEncryptionKey != "" {
		return []byte(c.JWTEncryptionKey)
	}

	if c.JWTEncryptionKeyFile != "" {
		key, err := os.ReadFile(c.JWTEncryptionKeyFile)
		if err == nil {
			c.JWTEncryptionKey = strings.TrimSpace(string(key))
			return []byte(c.JWTEncryptionKey)
		}
	}

	c.JWTEncryptionKey = rand.String(32, rand.GetAlphaNumericPool())

	return []byte(c.JWTEncryptionKey)
}
