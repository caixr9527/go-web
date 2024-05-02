package gateway

import "net/http"

// todo 接入注册中心
type GWConfig struct {
	Name        string
	Path        string
	Host        string
	Port        uint64
	Header      func(req *http.Request)
	ServiceName string
}
