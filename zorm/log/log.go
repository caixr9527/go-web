package log

import (
	"fmt"
	"io"
	"os"
	"path"
)

var (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Reset  = "\033[0m"
)

type LoggerLevel int

const (
	Debug LoggerLevel = iota
	Info
	Error
)

type Fields map[string]any

type Logger struct {
	Formatter    LoggingFormatter
	Level        LoggerLevel
	Outs         []*LoggerWriter
	LoggerFields Fields
	logPath      string
}

type LoggerWriter struct {
	Level LoggerLevel
	Out   io.Writer
}

type LoggingFormatter interface {
	Format(param *LoggingFormatParam) string
}

type LoggingFormatParam struct {
	Level        LoggerLevel
	IsColor      bool
	LoggerFields Fields
	Msg          any
}

type LoggerFormatter struct {
	Level        LoggerLevel
	IsColor      bool
	LoggerFields Fields
}

func (l LoggerLevel) Level() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Error:
		return "ERROR"
	default:
		return ""
	}
}

func New() *Logger {
	return &Logger{}
}

func Default() *Logger {
	logger := New()
	logger.Level = Debug
	w := &LoggerWriter{
		Level: Debug,
		Out:   os.Stdout,
	}
	logger.Outs = append(logger.Outs, w)
	logger.Formatter = &TextFormatter{}
	return logger
}

func (l *Logger) Info(msg any) {
	l.Print(msg, Info)
}

func (l *Logger) Debug(msg any) {
	l.Print(msg, Debug)
}

func (l *Logger) Error(msg any) {
	l.Print(msg, Error)
}
func (l *Logger) Print(msg any, level LoggerLevel) {
	if l.Level > level {
		return
	}

	param := &LoggingFormatParam{
		Level:        level,
		LoggerFields: l.LoggerFields,
		Msg:          msg,
	}
	str := l.Formatter.Format(param)
	for _, out := range l.Outs {
		if out.Out == os.Stdout {
			param.IsColor = true
			str = l.Formatter.Format(param)
			fmt.Fprintln(out.Out, str)
		}
		if out.Level == -1 || level == out.Level {
			fmt.Fprintln(out.Out, str)
		}
	}
}

func (l *Logger) WithFields(fields Fields) *Logger {
	return &Logger{
		Formatter:    l.Formatter,
		Outs:         l.Outs,
		Level:        l.Level,
		LoggerFields: fields,
	}
}

func (l *Logger) SetLogPath(logPath string) {
	l.logPath = logPath
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: -1,
		Out:   FileWrite(path.Join(logPath, "all.log")),
	})
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: Debug,
		Out:   FileWrite(path.Join(logPath, "debug.log")),
	})
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: Info,
		Out:   FileWrite(path.Join(logPath, "info.log")),
	})
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: Error,
		Out:   FileWrite(path.Join(logPath, "error.log")),
	})
}

func FileWrite(name string) io.Writer {
	w, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	return w
}
