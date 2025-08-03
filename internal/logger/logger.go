package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Config struct {
	Level  Level  `yaml:"level"`
	Format Format `yaml:"format"`
	Output string `yaml:"output"`
}

type Logger struct {
	*slog.Logger
}

var defaultLogger *Logger

//nolint:gochecknoinits // Global logger initialization is necessary
func init() {
	defaultLogger = New(Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: "stdout",
	})
}

func New(config Config) *Logger {
	var writer io.Writer
	switch config.Output {
	case "stderr":
		writer = os.Stderr
	case "stdout":
		writer = os.Stdout
	default:
		if config.Output != "" {
			file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				fmt.Fprintf(os.Stderr, "fail to open custom logger file. Using 'stdout' error: %s", err.Error())
				writer = os.Stdout
			} else {
				writer = file
			}
		} else {
			writer = os.Stdout
		}
	}

	var level slog.Level
	switch config.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, opts)
	case FormatText:
		fallthrough
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

func Default() *Logger {
	return defaultLogger
}

func SetDefault(logger *Logger) {
	defaultLogger = logger
}

//nolint:sloglint // Global convenience functions are intentional
func Debug(msg string, args ...interface{}) {
	defaultLogger.Debug(msg, args...)
}

//nolint:sloglint // Global convenience functions are intentional
func Info(msg string, args ...interface{}) {
	defaultLogger.Info(msg, args...)
}

//nolint:sloglint // Global convenience functions are intentional
func Warn(msg string, args ...interface{}) {
	defaultLogger.Warn(msg, args...)
}

//nolint:sloglint // Global convenience functions are intentional
func Error(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
}

//nolint:sloglint // Global convenience functions are intentional
func Fatal(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
	os.Exit(1)
}
