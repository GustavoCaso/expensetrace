package config

import (
	"fmt"
	"os"
	"time"

	"github.com/GustavoCaso/expensetrace/logger"
)

type Config struct {
	DBFile  string
	Logger  logger.Config
	Port    string
	Timeout time.Duration
}

const (
	defaultDBFile    = "expensetrace.db"
	defaultLogLevel  = logger.LevelInfo
	defaultLogFormat = logger.FormatText
	defaultLogOutput = "stdout"
	defaultPort      = "8080"
	defaultTimeout   = 5 * time.Second
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

	// Initialize configuration from environment variables
	if port := os.Getenv("EXPENSETRACE_PORT"); port != "" {
		c.Port = port
	} else {
		c.Port = defaultPort
	}

	if customTimeout := os.Getenv("EXPENSETRACE_TIMEOUT"); customTimeout != "" {
		duration, durationErr := time.ParseDuration(customTimeout)
		if durationErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse custom timeout, using default timeout of 5s")
			c.Timeout = defaultTimeout
		} else {
			c.Timeout = duration
		}
	} else {
		c.Timeout = defaultTimeout
	}
}

func Parse() *Config {
	conf := &Config{}

	conf.parseEnv()

	return conf
}
