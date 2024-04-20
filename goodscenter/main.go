package main

import (
	"github.com/caixr9527/goodscenter/model"
	"github.com/caixr9527/zorm"
	"net/http"
)

func main() {
	engine := zorm.Default()
	group := engine.Group("goods")
	group.Get("/find", func(ctx *zorm.Context) {
		goods := &model.Goods{
			Id:   1000,
			Name: "9002",
		}
		ctx.JSON(http.StatusOK, &model.Result{
			Code: 200,
			Msg:  "success",
			Data: goods,
		})
	})
	engine.Run(":9002")
}
