package service

import (
	"fmt"
	"github.com/caixr9527/zorm/orm"
	"net/url"
)

type User struct {
	Id       int64
	Username string
	Password string
}

func SaveUser() {
	dataSource := fmt.Sprintf("root@tcp(localhost:3306)/sys?charset=utf8&loc%sparseTime=true", url.QueryEscape("Asia/Shanghai"))
	zDb := orm.Open("mysql", dataSource)
	user := &User{
		Username: "smart",
		Password: "123456",
	}
	zDb.New().Table("User").Insert(user)
}
