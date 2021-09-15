package logging

import (
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
)

type LogLevel int

const (
	LogLevelNone LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelError
)

type LogEntry struct {
	Level   LogLevel
	Message string
	Fields  map[string]interface{}
}

func NewLogEntry(level LogLevel, message string, fields ...map[string]interface{}) LogEntry {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	return LogEntry{
		Level:   level,
		Message: message,
		Fields:  f,
	}
}

type LogHandler struct {
	entries chan LogEntry
}

func NewLogHandler() *LogHandler {
	h := &LogHandler{
		entries: make(chan LogEntry, 64),
	}
	go h.readEntries()
	return h
}

func (h *LogHandler) readEntries() {
	for entry := range h.entries {
		var l *logrus.Entry

		l = log.WithFields(entry.Fields)

		switch entry.Level {
		case LogLevelDebug:
			l.Debug(entry.Message)
		case LogLevelInfo:
			l.Info(entry.Message)
		case LogLevelError:
			l.Error(entry.Message)
		default:
			continue
		}
	}
}

func (h *LogHandler) handle(entry LogEntry) {
	select {
	case h.entries <- entry:
	default:
		return
	}
}

func (h *LogHandler) Log(entry LogEntry) {
	select {
	case h.entries <- entry:
	default:

	}
}
