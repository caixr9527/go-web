package main

import (
	"github.com/caixr9527/zorm"
	"github.com/caixr9527/zorm/gateway"
	"net/http"
)

func main() {
	engine := zorm.Default()
	engine.OpenGateway = true
	var configs []gateway.GWConfig
	configs = append(configs, gateway.GWConfig{
		Name: "order",
		Path: "/order/**",
		//Host: "127.0.0.1",
		//Port: 9003,
		Header: func(req *http.Request) {
			req.Header.Set("my", "caixiaorong")
		},
	}, gateway.GWConfig{
		Name: "goods",
		Path: "/goods/**",
		//Host: "127.0.0.1",
		//Port: 9002,
		Header: func(req *http.Request) {
			req.Header.Set("my", "caixiaorong")
		},
	})
	engine.SetGatewayConfig(configs)
	engine.Run(":81")
}
