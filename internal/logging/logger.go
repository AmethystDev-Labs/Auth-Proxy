package logging

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level     string
	Format    string
	AddSource bool
}

type AccessLogEntry struct {
	Method        string
	Path          string
	RouteKind     string
	Authenticated bool
	Status        int
	Duration      time.Duration
	ClientIP      string
}

type Logger interface {
	Info(module, requestID, message string, fields ...zap.Field)
	Warn(module, requestID, message string, fields ...zap.Field)
	Error(module, requestID, message string, fields ...zap.Field)
	Access(entry AccessLogEntry)
	Sync() error
}

type zapLogger struct {
	logger    *zap.Logger
	format    string
	addSource bool
}

type nopLogger struct{}

var requestCounter atomic.Uint64

func New(cfg Config, output io.Writer) (Logger, error) {
	if output == nil {
		output = io.Discard
	}

	level, err := zapcore.ParseLevel(strings.ToLower(defaultString(cfg.Level, "info")))
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	format := strings.ToLower(defaultString(cfg.Format, "text"))
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "ts",
		LevelKey:      "level",
		MessageKey:    "msg",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeTime:    zapcore.TimeEncoderOfLayout(time.RFC3339),
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		ConsoleSeparator: "  ",
	}

	var encoder zapcore.Encoder
	switch format {
	case "text":
		encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	default:
		return nil, fmt.Errorf("unsupported log format: %s", format)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(output), level)
	return &zapLogger{
		logger:    zap.New(core),
		format:    format,
		addSource: cfg.AddSource,
	}, nil
}

func Nop() Logger {
	return nopLogger{}
}

func (l *zapLogger) Info(module, requestID, message string, fields ...zap.Field) {
	l.log(zapcore.InfoLevel, module, requestID, message, 3, fields...)
}

func (l *zapLogger) Warn(module, requestID, message string, fields ...zap.Field) {
	l.log(zapcore.WarnLevel, module, requestID, message, 3, fields...)
}

func (l *zapLogger) Error(module, requestID, message string, fields ...zap.Field) {
	l.log(zapcore.ErrorLevel, module, requestID, message, 3, fields...)
}

func (l *zapLogger) Access(entry AccessLogEntry) {
	if l.format == "json" {
		l.logger.Info("http request",
			zap.String("method", entry.Method),
			zap.String("path", entry.Path),
			zap.String("route_kind", entry.RouteKind),
			zap.Bool("authenticated", entry.Authenticated),
			zap.Int("status", entry.Status),
			zap.Duration("duration", entry.Duration),
			zap.String("client_ip", entry.ClientIP),
			l.sourceField(3),
		)
		return
	}

	message := fmt.Sprintf("%-6s %-28s %-12s auth=%-3s %3d  %-8s %s",
		entry.Method,
		trimToWidth(entry.Path, 28),
		trimToWidth(defaultString(entry.RouteKind, "unknown"), 12),
		boolFlag(entry.Authenticated),
		entry.Status,
		entry.Duration.Truncate(time.Microsecond).String(),
		entry.ClientIP,
	)
	if source := l.sourceString(3); source != "" {
		message += "  " + source
	}
	l.logger.Info(message)
}

func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

func (l *zapLogger) log(level zapcore.Level, module, requestID, message string, callerSkip int, fields ...zap.Field) {
	if l.format == "json" {
		allFields := make([]zap.Field, 0, len(fields)+3)
		if module != "" {
			allFields = append(allFields, zap.String("module", module))
		}
		if requestID != "" {
			allFields = append(allFields, zap.String("request_id", requestID))
		}
		allFields = append(allFields, fields...)
		if sourceField := l.sourceField(callerSkip); sourceField.Key != "" {
			allFields = append(allFields, sourceField)
		}

		switch level {
		case zapcore.InfoLevel:
			l.logger.Info(message, allFields...)
		case zapcore.WarnLevel:
			l.logger.Warn(message, allFields...)
		default:
			l.logger.Error(message, allFields...)
		}
		return
	}

	formatted := formatTextMessage(module, requestID, message)
	if source := l.sourceString(callerSkip); source != "" {
		formatted += "  " + source
	}

	switch level {
	case zapcore.InfoLevel:
		l.logger.Info(formatted)
	case zapcore.WarnLevel:
		l.logger.Warn(formatted)
	default:
		l.logger.Error(formatted)
	}
}

func (l *zapLogger) sourceField(callerSkip int) zap.Field {
	if source := l.sourceString(callerSkip); source != "" {
		return zap.String("source", source)
	}
	return zap.Skip()
}

func (l *zapLogger) sourceString(callerSkip int) string {
	if !l.addSource {
		return ""
	}

	_, file, line, ok := runtime.Caller(callerSkip)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

func (nopLogger) Info(string, string, string, ...zap.Field)  {}
func (nopLogger) Warn(string, string, string, ...zap.Field)  {}
func (nopLogger) Error(string, string, string, ...zap.Field) {}
func (nopLogger) Access(AccessLogEntry)                      {}
func (nopLogger) Sync() error                                { return nil }

func NextRequestID() string {
	value := requestCounter.Add(1)
	return fmt.Sprintf("%08x", value)
}

func formatTextMessage(module, requestID, message string) string {
	parts := make([]string, 0, 3)
	if module != "" {
		parts = append(parts, fmt.Sprintf("[%s]", module))
	}
	if requestID != "" {
		parts = append(parts, fmt.Sprintf("[%s]", requestID))
	}
	if message != "" {
		parts = append(parts, message)
	}
	return strings.Join(parts, "  ")
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func boolFlag(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func trimToWidth(value string, width int) string {
	if len(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "…"
}
