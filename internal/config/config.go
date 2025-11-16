package config

import (
	"os"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type Config struct {
	DBFile string
	Logger logger.Config
}

const (
	defaultDBFile    = "expensetrace.db"
	defaultLogLevel  = logger.LevelInfo
	defaultLogFormat = logger.FormatText
	defaultLogOutput = "stdout"
)

func (c *Config) parseEnv() {
	if db := os.Getenv("EXPENSETRACE_DB"); db != "" {
		c.DBFile = db
	} else {
		c.DBFile = defaultDBFile
	}

	if level := os.Getenv("EXPENSETRACE_LOG_LEVEL"); level != "" {
		c.Logger.Level = logger.Level(level)
	} else {
		c.Logger.Level = defaultLogLevel
	}

	if format := os.Getenv("EXPENSETRACE_LOG_FORMAT"); format != "" {
		c.Logger.Format = logger.Format(format)
	} else {
		c.Logger.Format = defaultLogFormat
	}

	if output := os.Getenv("EXPENSETRACE_LOG_OUTPUT"); output != "" {
		c.Logger.Output = output
	} else {
		c.Logger.Output = defaultLogOutput
	}
}

func Parse() *Config {
	conf := &Config{}

	conf.parseEnv()

	return conf
}
