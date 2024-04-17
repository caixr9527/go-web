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
	id, _, err := zDb.New(user).Table("User").Insert(user)
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	zDb.Close()

}

func SaveUserBatch() {
	//dataSource := fmt.Sprintf("root:root@tcp(localhost:3306)/sys?charset=utf8&loc%sparseTime=true", url.QueryEscape("Asia/Shanghai"))
	//dataSource := fmt.Sprintf("root:root@tcp(localhost:3306)/sys?charset=utf8&loc%sparseTime=true", url.QueryEscape("Asia/Shanghai"))
	zDb := orm.Open("mysql", "root:root@tcp(localhost:3306)/sys?charset=utf8")
	//zDb.Prefix()
	user := &User{
		//Id:       1,
		UserName: "smart22",
		Password: "123456",
	}
	user1 := &User{
		//Id:       1,
		UserName: "smart11",
		Password: "123456",
	}
	var users []any
	users = append(users, user1, user)
	id, _, err := zDb.New(user).Table("User").InsertBatch(users)
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	zDb.Close()

}

func UpdateUser() {
	//dataSource := fmt.Sprintf("root:root@tcp(localhost:3306)/sys?charset=utf8&loc%sparseTime=true", url.QueryEscape("Asia/Shanghai"))
	//dataSource := fmt.Sprintf("root:root@tcp(localhost:3306)/sys?charset=utf8&loc%sparseTime=true", url.QueryEscape("Asia/Shanghai"))
	zDb := orm.Open("mysql", "root:root@tcp(localhost:3306)/sys?charset=utf8")
	//zDb.Prefix()
	user := &User{
		//Id:       1,
		UserName: "smart66699",
		Password: "123456",
	}
	fmt.Println(user)
	//id, _, err := zDb.New().Table("User").Where("id", 1).Update("user_name", "smart666")
	//id, _, err := zDb.New().Table("User").Where("id", 1).Update(user)
	id, _, err := zDb.New(user).Table("User").
		Where("id", 1).
		UpdateParam("password", 1111).
		Update()
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	zDb.Close()

}

func SelectOne() {
	zDb := orm.Open("mysql", "root:root@tcp(localhost:3306)/sys?charset=utf8")
	//zDb.Prefix()
	user := &User{}
	fmt.Println(user)
	err := zDb.New(user).Table("User").
		Where("id", 2).
		SelectOne(user)
	fmt.Println(user)
	if err != nil {
		panic(err)
	}
	zDb.Close()

}

func Select() {
	zDb := orm.Open("mysql", "root:root@tcp(localhost:3306)/sys?charset=utf8")
	//zDb.Prefix()
	user := &User{}
	fmt.Println(user)
	users, err := zDb.New(user).Table("User").
		Select(user)
	for _, v := range users {
		u := v.(*User)
		fmt.Println(u)
	}
	fmt.Println(users)
	if err != nil {
		panic(err)
	}
	zDb.Close()

}
