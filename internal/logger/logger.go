package logger

import (
	"context"
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
			file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
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

func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{Logger: l.Logger.With()}
}

func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{Logger: l.Logger.With("component", component)}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{Logger: l.Logger.With(args...)}
}

func Debug(msg string, args ...interface{}) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
}

func Fatal(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
	os.Exit(1)
}

func DebugCtx(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.DebugContext(ctx, msg, args...)
}

func InfoCtx(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.InfoContext(ctx, msg, args...)
}

func WarnCtx(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.WarnContext(ctx, msg, args...)
}

func ErrorCtx(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.ErrorContext(ctx, msg, args...)
}