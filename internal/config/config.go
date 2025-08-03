package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type Category struct {
	Name    string `yaml:"name"`
	Pattern string `yaml:"pattern"`
}
type Categories struct {
	Expense []Category `yaml:"expense"`
	Income  []Category `yaml:"income"`
}

type Config struct {
	DB         string        `yaml:"db"`
	Categories Categories    `yaml:"categories"`
	Logger     logger.Config `yaml:"logger"`
}

func (c *Config) parseEnv() {
	if c.DB == "" {
		c.DB = os.Getenv("EXPENSETRACE_DB")
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

func (c *Config) validate() error {
	if c.DB == "" {
		return errors.New("DB is not set")
	}

	return nil
}

func Parse(file string) (*Config, error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = yaml.Unmarshal(bytes, conf)
	if err != nil {
		return nil, err
	}

	conf.parseEnv()

	err = conf.validate()
	if err != nil {
		return nil, err
	}

	return conf, nil
}
