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
	DB     DBConfig      `yaml:"db"`
	Logger logger.Config `yaml:"logger"`
}

func (c *Config) parseEnv() {
	// Database source path
	if c.DB.Source == "" {
		if db := os.Getenv("EXPENSETRACE_DB"); db != "" {
			c.DB.Source = db
		} else {
			c.DB.Source = "expensetrace.db"
		}
	}

	// Connection pool settings
	if c.DB.MaxOpenConns == 0 {
		if maxOpenConns := os.Getenv("EXPENSETRACE_DB_MAX_OPEN_CONNS"); maxOpenConns != "" {
			if val, err := strconv.Atoi(maxOpenConns); err == nil {
				c.DB.MaxOpenConns = val
			}
		}
	}

	if c.DB.MaxIdleConns == 0 {
		if maxIdleConns := os.Getenv("EXPENSETRACE_DB_MAX_IDLE_CONNS"); maxIdleConns != "" {
			if val, err := strconv.Atoi(maxIdleConns); err == nil {
				c.DB.MaxIdleConns = val
			}
		}
	}

	if c.DB.ConnMaxLifetime == 0 {
		if connMaxLifetime := os.Getenv("EXPENSETRACE_DB_CONN_MAX_LIFETIME"); connMaxLifetime != "" {
			if val, err := time.ParseDuration(connMaxLifetime); err == nil {
				c.DB.ConnMaxLifetime = val
			}
		}
	}

	if c.DB.ConnMaxIdleTime == 0 {
		if connMaxIdleTime := os.Getenv("EXPENSETRACE_DB_CONN_MAX_IDLE_TIME"); connMaxIdleTime != "" {
			if val, err := time.ParseDuration(connMaxIdleTime); err == nil {
				c.DB.ConnMaxIdleTime = val
			}
		}
	}

	// SQLite PRAGMA settings
	if c.DB.JournalMode == "" {
		if journalMode := os.Getenv("EXPENSETRACE_DB_JOURNAL_MODE"); journalMode != "" {
			c.DB.JournalMode = journalMode
		}
	}

	if c.DB.Synchronous == "" {
		if synchronous := os.Getenv("EXPENSETRACE_DB_SYNCHRONOUS"); synchronous != "" {
			c.DB.Synchronous = synchronous
		}
	}

	if c.DB.CacheSize == 0 {
		if cacheSize := os.Getenv("EXPENSETRACE_DB_CACHE_SIZE"); cacheSize != "" {
			if val, err := strconv.Atoi(cacheSize); err == nil {
				c.DB.CacheSize = val
			}
		}
	}

	if c.DB.BusyTimeout == 0 {
		if busyTimeout := os.Getenv("EXPENSETRACE_DB_BUSY_TIMEOUT"); busyTimeout != "" {
			if val, err := strconv.Atoi(busyTimeout); err == nil {
				c.DB.BusyTimeout = val
			}
		}
	}

	if c.DB.WALAutocheckpoint == 0 {
		if walAutocheckpoint := os.Getenv("EXPENSETRACE_DB_WAL_AUTOCHECKPOINT"); walAutocheckpoint != "" {
			if val, err := strconv.Atoi(walAutocheckpoint); err == nil {
				c.DB.WALAutocheckpoint = val
			}
		}
	}

	if c.DB.TempStore == "" {
		if tempStore := os.Getenv("EXPENSETRACE_DB_TEMP_STORE"); tempStore != "" {
			c.DB.TempStore = tempStore
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
