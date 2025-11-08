package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type DBConfig struct {
	// Database file path
	Source string `yaml:"source"`

	// Connection pool settings
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`

	// SQLite PRAGMA settings
	JournalMode      string `yaml:"journal_mode"`
	Synchronous      string `yaml:"synchronous"`
	CacheSize        int    `yaml:"cache_size"`
	BusyTimeout      int    `yaml:"busy_timeout"`
	WALAutocheckpoint int   `yaml:"wal_autocheckpoint"`
	TempStore        string `yaml:"temp_store"`
}

type Config struct {
	DB     string        `yaml:"db"`
	Logger logger.Config `yaml:"logger"`
}

func (c *Config) parseEnv() {
	// Database source path
	if c.DB == "" {
		if db := os.Getenv("EXPENSETRACE_DB"); db != "" {
			c.DB = db
		} else {
			c.DB = "expensetrace.db"
		}
	}

	// Logger settings
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

// GetDBConfig builds a DBConfig from the database path and environment variables
func (c *Config) GetDBConfig() DBConfig {
	dbConfig := DBConfig{
		Source: c.DB,
	}

	// Connection pool settings from environment variables
	if maxOpenConns := os.Getenv("EXPENSETRACE_DB_MAX_OPEN_CONNS"); maxOpenConns != "" {
		if val, err := strconv.Atoi(maxOpenConns); err == nil {
			dbConfig.MaxOpenConns = val
		}
	}

	if maxIdleConns := os.Getenv("EXPENSETRACE_DB_MAX_IDLE_CONNS"); maxIdleConns != "" {
		if val, err := strconv.Atoi(maxIdleConns); err == nil {
			dbConfig.MaxIdleConns = val
		}
	}

	if connMaxLifetime := os.Getenv("EXPENSETRACE_DB_CONN_MAX_LIFETIME"); connMaxLifetime != "" {
		if val, err := time.ParseDuration(connMaxLifetime); err == nil {
			dbConfig.ConnMaxLifetime = val
		}
	}

	if connMaxIdleTime := os.Getenv("EXPENSETRACE_DB_CONN_MAX_IDLE_TIME"); connMaxIdleTime != "" {
		if val, err := time.ParseDuration(connMaxIdleTime); err == nil {
			dbConfig.ConnMaxIdleTime = val
		}
	}

	// SQLite PRAGMA settings from environment variables
	if journalMode := os.Getenv("EXPENSETRACE_DB_JOURNAL_MODE"); journalMode != "" {
		dbConfig.JournalMode = journalMode
	}

	if synchronous := os.Getenv("EXPENSETRACE_DB_SYNCHRONOUS"); synchronous != "" {
		dbConfig.Synchronous = synchronous
	}

	if cacheSize := os.Getenv("EXPENSETRACE_DB_CACHE_SIZE"); cacheSize != "" {
		if val, err := strconv.Atoi(cacheSize); err == nil {
			dbConfig.CacheSize = val
		}
	}

	if busyTimeout := os.Getenv("EXPENSETRACE_DB_BUSY_TIMEOUT"); busyTimeout != "" {
		if val, err := strconv.Atoi(busyTimeout); err == nil {
			dbConfig.BusyTimeout = val
		}
	}

	if walAutocheckpoint := os.Getenv("EXPENSETRACE_DB_WAL_AUTOCHECKPOINT"); walAutocheckpoint != "" {
		if val, err := strconv.Atoi(walAutocheckpoint); err == nil {
			dbConfig.WALAutocheckpoint = val
		}
	}

	if tempStore := os.Getenv("EXPENSETRACE_DB_TEMP_STORE"); tempStore != "" {
		dbConfig.TempStore = tempStore
	}

	return dbConfig
}
