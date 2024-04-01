package zorm

import (
	"log"
	"net"
	"strings"
	"time"
)

type LoggingConfig struct {
}

func LoggingWithConfig(conf LoggingConfig, next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		r := ctx.R
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

		log.Printf("[zorm] %v | %3d | %13v | %15v | %-7s %#v\n",
			stop.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency, clientIp, method, path)
	}
}
func Logging(next HandlerFunc) HandlerFunc {
	return LoggingWithConfig(LoggingConfig{}, next)
}
