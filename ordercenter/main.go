package main

import (
	"github.com/caixr9527/zorm"
	"github.com/caixr9527/zorm/rpc"
	"net/http"
)

func main() {
	engine := zorm.Default()
	client := rpc.NewHttpClient()
	group := engine.Group("order")
	group.Get("/find", func(ctx *zorm.Context) {
		params := make(map[string]any)
		params["id"] = 1000
		params["name"] = "1000"
		body, err := client.Get("http://localhost:9002/goods/find", params)
		if err != nil {
			panic(err)
		}
		ctx.JSON(http.StatusOK, string(body))
	})
	engine.Run(":9003")
}
