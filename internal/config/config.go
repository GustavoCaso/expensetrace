package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type Config struct {
	DB     string        `yaml:"db"`
	Logger logger.Config `yaml:"logger"`
}

func (c *Config) parseEnv() {
	if c.DB == "" {
		if db := os.Getenv("EXPENSETRACE_DB"); db != "" {
			c.DB = db
		} else {
			c.DB = "expensetrace.db"
		}
	}

	if c.Logger.Level == "" {
		if level := os.Getenv("EXPENSETRACE_LOG_LEVEL"); level != "" {
			c.Logger.Level = logger.Level(level)
		} else {
			c.Logger.Level = logger.LevelInfo
		}
	}

	if c.Logger.Format == "" {
		if format := os.Getenv("EXPENSETRACE_LOG_FORMAT"); format != "" {
			c.Logger.Format = logger.Format(format)
		} else {
			c.Logger.Format = logger.FormatText
		}
	}

	if c.Logger.Output == "" {
		if output := os.Getenv("EXPENSETRACE_LOG_OUTPUT"); output != "" {
			c.Logger.Output = output
		} else {
			c.Logger.Output = "stdout"
		}
	}
}

func Parse(file string) (*Config, error) {
	conf := &Config{}

	_, statErr := os.Stat(file)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return nil, statErr
	}

	if statErr == nil {
		bytes, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(bytes, conf)
		if err != nil {
			return nil, err
		}
	}

	conf.parseEnv()

	return conf, nil
}
