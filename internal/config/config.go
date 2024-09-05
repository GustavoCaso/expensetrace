package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Category struct {
	Name    string
	Pattern string
}

type Config struct {
	DB         string
	Categories []Category
}

func Parse(file string) (*Config, error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = yaml.Unmarshal(bytes, &conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}
