package main

import (
	"errors"
	"github.com/caixr9527/goodscenter/model"
	"github.com/caixr9527/zorm"
	"github.com/caixr9527/zorm/breaker"
	"net/http"
)

func main() {
	engine := zorm.Default()
	//engine.Use(zorm.Limiter(1, 1))
	group := engine.Group("goods")
	settings := breaker.Settings{}
	settings.Fallback = func(err error) (any, error) {
		goods := &model.Goods{
			Id:   1000,
			Name: "降级",
		}
		return goods, err
	}
	var cb = breaker.NewCircuitBreaker(settings)
	group.Get("/find", func(ctx *zorm.Context) {
		result, err := cb.Execute(func() (any, error) {
			query := ctx.GetQuery("id")
			if query == "2" {
				return nil, errors.New("测试短路")
			}
			//fmt.Println(ctx.GetHeader("my"))
			goods := &model.Goods{
				Id:   1000,
				Name: "9002",
			}

			return goods, nil
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, &model.Result{
				Code: 500,
				Msg:  err.Error(),
				Data: result,
			})
			return
		}
		ctx.JSON(http.StatusOK, &model.Result{
			Code: 200,
			Msg:  "success",
			Data: result,
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
	//tcpServer, err := rpc.NewTcpServer("127.0.0.1", 9222)
	//log.Println(err)
	//gob.Register(&model.Result{})
	//gob.Register(&model.Goods{})
	//tcpServer.Register("goods", &service.GoodsRpcService{})
	//tcpServer.Run()
	engine.Run(":9002")

}
