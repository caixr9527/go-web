package service

import "github.com/caixr9527/zorm/rpc"

type GoodsService struct {
	Find func(args map[string]any) ([]byte, error) `zrpc:"GET,/goods/find"`
}

func (g *GoodsService) Env() rpc.HttpConfig {
	return rpc.HttpConfig{
		Host: "localhost",
		Port: 9002,
	}
}
