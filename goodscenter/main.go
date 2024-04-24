package main

import (
	"github.com/caixr9527/goodscenter/model"
	"github.com/caixr9527/ordercenter/service"
	"github.com/caixr9527/zorm"
	"github.com/caixr9527/zorm/rpc"
	"log"
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
	//listen, _ := net.Listen("tcp", ":9111")
	//server := grpc.NewServer()
	//server, _ := rpc.NewGrpcServer(":9111")
	//server.Register(func(g *grpc.Server) {
	//	api.RegisterGoodsApiServer(g, &api.GoodsRpcService{})
	//})
	//err := server.Run()
	//log.Println(err)
	tcpServer, err := rpc.NewTcpServer(":9222")
	log.Println(err)
	tcpServer.Register("goods", &service.GoodsService{})
	tcpServer.Run()
	engine.Run(":9002")
}
