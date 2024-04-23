package api

import (
	"context"
)

type GoodsRpcService struct {
}

func (GoodsRpcService) Find(ctx context.Context, request *GoodsRequest) (*GoodsResponse, error) {
	goods := &Goods{
		Id:   1000,
		Name: "hhhh",
	}
	res := &GoodsResponse{
		Code: 200,
		Msg:  "success",
		Data: goods,
	}
	return res, nil
}

func (GoodsRpcService) mustEmbedUnimplementedGoodsApiServer() {}
