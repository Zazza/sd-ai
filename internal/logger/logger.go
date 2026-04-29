package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Level string

const (
	ERROR Level = "error"
	WARN  Level = "warn"
	INFO  Level = "info"
	DEBUG Level = "debug"
)

type Entry struct {
	Level     Level  `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type Logger struct {
	ctx context.Context
	mu  sync.Mutex
}

func New(ctx context.Context) *Logger {
	return &Logger{ctx: ctx}
}

func (l *Logger) SetContext(ctx context.Context) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ctx = ctx
}

func (l *Logger) emit(level Level, format string, args ...interface{}) {
	e := Entry{
		Level:     level,
		Message:   fmt.Sprintf(format, args...),
		Timestamp: time.Now().Format("15:04:05"),
	}

	data, _ := json.Marshal(e)

	l.mu.Lock()
	ctx := l.ctx
	l.mu.Unlock()

	if ctx != nil {
		runtime.EventsEmit(ctx, "log:entry", string(data))
	}
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.emit(ERROR, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.emit(WARN, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.emit(INFO, format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.emit(DEBUG, format, args...)
}

func (l *Logger) APIRequest(method, url string) {
	l.Debug("API %s %s", method, shortURL(url))
}

func (l *Logger) APIResponse(method, url string, status int) {
	if status >= 400 {
		l.emit(ERROR, "API %s %s -> %d", method, shortURL(url), status)
	} else {
		l.emit(DEBUG, "API %s %s -> %d", method, shortURL(url), status)
	}
}

func (l *Logger) UserAction(action string, args ...interface{}) {
	l.Info(action, args...)
}

func shortURL(url string) string {
	if i := strings.Index(url, "/sdapi/"); i >= 0 {
		return url[i:]
	}
	if i := strings.Index(url, "/v1/"); i >= 0 {
		return url[i:]
	}
	if i := strings.Index(url, "/api/"); i >= 0 {
		return url[i:]
	}
	return url
}
