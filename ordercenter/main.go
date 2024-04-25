package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"github.com/caixr9527/goodscenter/model"
	"github.com/caixr9527/ordercenter/api"
	"github.com/caixr9527/ordercenter/service"
	"github.com/caixr9527/zorm"
	"github.com/caixr9527/zorm/rpc"
	"log"
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

	group.Get("/findTcp", func(ctx *zorm.Context) {
		gob.Register(&model.Result{})
		gob.Register(&model.Goods{})
		option := rpc.DefaultOption
		option.SerializerType = rpc.Gob
		proxy := rpc.NewTcpClientProxy(option)
		params := make([]any, 1)
		params[0] = int64(1)
		result, err := proxy.Call(context.Background(), "goods", "Find", params)
		log.Println(err)
		ctx.JSON(http.StatusOK, result)
	})
	engine.Run(":9003")
}
