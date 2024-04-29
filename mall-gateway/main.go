package main

import (
	"github.com/caixr9527/zorm"
	"github.com/caixr9527/zorm/gateway"
)

func main() {
	engine := zorm.Default()
	engine.OpenGateway = true
	var configs []gateway.GWConfig
	configs = append(configs, gateway.GWConfig{
		Name: "order",
		Path: "/order/**",
		Host: "127.0.0.1",
		Port: 9003,
	}, gateway.GWConfig{
		Name: "goods",
		Path: "/goods/**",
		Host: "127.0.0.1",
		Port: 9002,
	})
	engine.SetGatewayConfig(configs)
	engine.Run(":81")
}
