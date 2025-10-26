package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/fatih/color"
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
		// Text format to file, colorized to stdout only if it's a terminal
		fileHandler := slog.NewTextHandler(fileWriter, &slog.HandlerOptions{Level: level})

		var stdoutHandler slog.Handler
		// Only use colors if stdout is a terminal (not redirected/piped)
		if isTerminal(os.Stdout) {
			stdoutHandler = NewColorHandler(os.Stdout, &slog.HandlerOptions{Level: level})
		} else {
			stdoutHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
		}

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

// ColorHandler wraps slog.Handler to add color output
type ColorHandler struct {
	handler slog.Handler
}

func NewColorHandler(w io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	return &ColorHandler{
		handler: slog.NewTextHandler(w, opts),
	}
}

func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	// Colorize based on level
	var colorFunc func(string, ...interface{}) string
	switch r.Level {
	case slog.LevelDebug:
		colorFunc = color.CyanString
	case slog.LevelInfo:
		colorFunc = color.GreenString
	case slog.LevelWarn:
		colorFunc = color.YellowString
	case slog.LevelError:
		colorFunc = color.RedString
	default:
		colorFunc = color.WhiteString
	}

	r.Message = colorFunc(r.Message)
	return h.handler.Handle(ctx, r)
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColorHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return &ColorHandler{handler: h.handler.WithGroup(name)}
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
