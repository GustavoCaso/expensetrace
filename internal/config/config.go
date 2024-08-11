package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DB         string
	Categories map[string]string
}

func Parse(file string) (*Config, error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = toml.Unmarshal(bytes, &conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}
