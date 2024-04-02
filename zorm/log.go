package zorm

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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

var DefaultWriter io.Writer = os.Stdout

type LoggingConfig struct {
	Formatter LoggerFormatter
	out       io.Writer
}

type LoggerFormatter = func(params *LogFormatterParams) string

type LogFormatterParams struct {
	Request        *http.Request
	TimeStamp      time.Time
	StatusCode     int
	Latency        time.Duration
	ClientIp       net.IP
	Method         string
	Path           string
	IsDisplayColor bool
}

func (p LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode
	switch code {
	case http.StatusOK:
		return Green
	default:
		return Red
	}
}

func (p LogFormatterParams) ResetColor() string {
	return Reset
}

var defaultFormatter = func(params *LogFormatterParams) string {
	var statusCodeColor = params.StatusCodeColor()
	var reset = params.ResetColor()
	if params.Latency > time.Minute {
		params.Latency = params.Latency.Truncate(time.Second)
	}
	if params.IsDisplayColor {
		return fmt.Sprintf("[zorm] %v |%s %3d %s| %13v | %15v | %-7s %#v\n",
			params.TimeStamp.Format("2006/01/02 - 15:04:05"),
			statusCodeColor, params.StatusCode, reset,
			params.Latency, params.ClientIp, params.Method, params.Path)
	}
	return fmt.Sprintf("[zorm] %v | %3d | %13v | %15v | %-7s %#v\n",
		params.TimeStamp.Format("2006/01/02 - 15:04:05"),
		params.StatusCode,
		params.Latency, params.ClientIp, params.Method, params.Path)
}

func LoggingWithConfig(conf LoggingConfig, next HandlerFunc) HandlerFunc {
	formatter := conf.Formatter
	if formatter == nil {
		formatter = defaultFormatter
	}
	out := conf.out
	displayColor := false
	if out == nil {
		out = DefaultWriter
		displayColor = true
	}

	return func(ctx *Context) {
		r := ctx.R
		param := &LogFormatterParams{
			Request:        r,
			IsDisplayColor: displayColor,
		}
		start := time.Now()
		path := r.URL.Path
		raw := r.URL.RawQuery

		next(ctx)

		stop := time.Now()
		latency := stop.Sub(start)
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
		clientIp := net.ParseIP(ip)
		method := r.Method
		statusCode := ctx.StatusCode

		if raw != "" {
			path = path + "?" + raw
		}

		param.TimeStamp = stop
		param.Latency = latency
		param.ClientIp = clientIp
		param.Method = method
		param.StatusCode = statusCode
		param.Path = path

		_, _ = fmt.Fprint(out, formatter(param))
	}
}
func Logging(next HandlerFunc) HandlerFunc {
	return LoggingWithConfig(LoggingConfig{}, next)
}
