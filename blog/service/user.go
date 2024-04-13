package service

import (
	"fmt"
	"github.com/caixr9527/zorm/orm"
	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	Id       int64  //`zorm:"id,auto_increment"`
	UserName string //`zorm:"user_name"`
	Password string //`zorm:"password"`
}

func SaveUser() {
	//dataSource := fmt.Sprintf("root:root@tcp(localhost:3306)/sys?charset=utf8&loc%sparseTime=true", url.QueryEscape("Asia/Shanghai"))
	//dataSource := fmt.Sprintf("root:root@tcp(localhost:3306)/sys?charset=utf8&loc%sparseTime=true", url.QueryEscape("Asia/Shanghai"))
	zDb := orm.Open("mysql", "root:root@tcp(localhost:3306)/sys?charset=utf8")
	//zDb.Prefix()
	user := &User{
		//Id:       1,
		UserName: "smart",
		Password: "123456",
	}
	id, _, err := zDb.New().Table("User").Insert(user)
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	zDb.Close()

}
