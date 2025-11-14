package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger(cfg config.LoggingConfig) *slog.Logger {
	// File writer with rotation
	fileWriter := &lumberjack.Logger{
		Filename:   cfg.OutputFile,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   true,
	}

	// Determine log level
	level := parseLevel(cfg.Level)

	// Create handlers
	var handler slog.Handler

	if cfg.Format == "json" {
		// JSON format to both stdout and file
		multiWriter := io.MultiWriter(os.Stdout, fileWriter)
		handler = slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		// Text format to file (plain), tinted/colored to stdout (if terminal supports it)
		fileHandler := slog.NewTextHandler(fileWriter, &slog.HandlerOptions{Level: level})

		// Use tint for colored console output
		// tint automatically handles color detection based on terminal capabilities
		stdoutHandler := tint.NewHandler(os.Stdout, &tint.Options{
			Level:   level,
			NoColor: !shouldUseColors(),
		})

		handler = NewMultiHandler(stdoutHandler, fileHandler)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// isTerminal checks if the given file is a terminal (TTY)
func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// shouldUseColors determines if colored output should be used
// based on terminal capabilities and environment settings
func shouldUseColors() bool {
	// Check if stdout is a terminal
	if !isTerminal(os.Stdout) {
		return false
	}

	// Respect NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Don't use colors for dumb terminals
	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false
	}

	return true
}

// MultiHandler writes to multiple handlers
type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: newHandlers}
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}
