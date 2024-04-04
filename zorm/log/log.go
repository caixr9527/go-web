package log

import (
	"fmt"
	"github.com/caixr9527/zorm/internal/zstring"
	"io"
	"log"
	"os"
	"path"
	"strings"
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

type Fields map[string]any

type Logger struct {
	Formatter    LoggingFormatter
	Level        LoggerLevel
	Outs         []*LoggerWriter
	LoggerFields Fields
	logPath      string
	LogFileSize  int64
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
			l.checkFileSize(out)
		}
		if out.Level == -1 || level == out.Level {
			fmt.Fprintln(out.Out, str)
			l.checkFileSize(out)
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

func (l *Logger) checkFileSize(writer *LoggerWriter) {
	logFile := writer.Out.(*os.File)
	if logFile != nil {
		stat, err := logFile.Stat()
		if err != nil {
			log.Println(err)
			return
		}
		size := stat.Size()
		if l.LogFileSize <= 0 {
			l.LogFileSize = 100 << 20
		}
		if size >= l.LogFileSize {
			// todo 需要优化，应该一直往info.log文件里面写，满了再归档到另一个文件下
			// todo 可添加，按天归档
			_, name := path.Split(stat.Name())
			fileName := name[0:strings.Index(name, ".")]
			write := FileWrite(path.Join(l.logPath, zstring.JoinStrings(fileName, ".", time.Now().UnixMilli(), ".log")))
			writer.Out = write
		}
	}
}

func FileWrite(name string) io.Writer {
	w, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	return w
}
