package service

import "github.com/caixr9527/goodscenter/model"

type GoodsRpcService struct {
}

func (*GoodsRpcService) Find(id int64) *model.Result {
	goods := model.Goods{
		Id:   id,
		Name: "goods center",
	}
	return &model.Result{
		Code: 200,
		Msg:  "success",
		Data: goods,
	}
}
