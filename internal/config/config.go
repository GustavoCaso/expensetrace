package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Category struct {
	Name    string `yaml:"name"`
	Pattern string `yaml:"pattern"`
}

type Config struct {
	DB         string     `yaml:"db"`
	Categories []Category `yaml:"categories"`
}

func (c *Config) parseEnv() {
	if c.DB == "" {
		c.DB = os.Getenv("EXPENSETRACE_DB")
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
