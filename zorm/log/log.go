package log

import (
	"fmt"
	"io"
	"os"
	"time"
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

type Logger struct {
	Formatter LoggerFormatter
	Level     LoggerLevel
	Outs      []io.Writer
}

type LoggerFormatter struct {
	Level   LoggerLevel
	IsColor bool
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
	logger.Outs = append(logger.Outs, os.Stdout)
	logger.Formatter = LoggerFormatter{}
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
	l.Formatter.Level = level
	str := l.Formatter.format(msg)
	for _, out := range l.Outs {
		if out == os.Stdout {
			l.Formatter.IsColor = true
			str = l.Formatter.format(msg)
		}
		fmt.Fprintln(out, str)
	}
}

func (f *LoggerFormatter) format(msg any) string {
	now := time.Now()
	if f.IsColor {
		levelColor := f.LevelColor()
		msgColor := f.MsgColor()
		return fmt.Sprintf("[zorm] | %s [%s] %s | %v  | %s %#v %s",
			levelColor, f.Level.Level(), Reset,
			now.Format("2006/01/02 - 15:04:05"),
			msgColor, msg, Reset)
	}
	return fmt.Sprintf("[zorm] | [%s] | %v  | msg=%#v",
		f.Level.Level(),
		now.Format("2006/01/02 - 15:04:05"),
		msg)
}

func (f *LoggerFormatter) LevelColor() string {
	switch f.Level {
	case Debug:
		return Blue
	case Info:
		return Green
	case Error:
		return Red
	default:
		return ""
	}
}

func (f *LoggerFormatter) MsgColor() string {
	switch f.Level {
	case Debug:
		return Blue
	case Info:
		return Green
	case Error:
		return Red
	default:
		return ""
	}
}
