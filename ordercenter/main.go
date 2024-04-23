package main

import (
	"context"
	"encoding/json"
	"github.com/caixr9527/goodscenter/model"
	"github.com/caixr9527/ordercenter/api"
	"github.com/caixr9527/ordercenter/service"
	"github.com/caixr9527/zorm"
	"github.com/caixr9527/zorm/rpc"
	"net/http"
)

func main() {
	engine := zorm.Default()
	client := rpc.NewHttpClient()
	client.RegisterHttpService("goods", &service.GoodsService{})
	group := engine.Group("order")
	group.Get("/find", func(ctx *zorm.Context) {
		params := make(map[string]any)
		params["id"] = 1000
		params["name"] = "1000"
		//body, err := client.Get("http://localhost:9002/goods/find", params)
		//if err != nil {
		//	panic(err)
		//}

		body, err := client.Do("goods", "Find").(*service.GoodsService).Find(params)
		if err != nil {
			panic(err)
		}
		v := &model.Result{}
		json.Unmarshal(body, v)
		ctx.JSON(http.StatusOK, v)
	})
	group.Get("/findGrpc", func(ctx *zorm.Context) {
		config := rpc.DefaultGrpcClientConfig()
		config.Address = "localhost:9111"
		client, err := rpc.NewGrpcClient(config)
		if err != nil {
			panic(err)
		}
		defer client.Conn.Close()
		apiClient := api.NewGoodsApiClient(client.Conn)
		goodsResponse, err := apiClient.Find(context.Background(), &api.GoodsRequest{})
		ctx.JSON(http.StatusOK, goodsResponse)
	})
	engine.Run(":9003")
}
