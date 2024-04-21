package main

import (
	"encoding/json"
	"github.com/caixr9527/goodscenter/model"
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
	engine.Run(":9003")
}
