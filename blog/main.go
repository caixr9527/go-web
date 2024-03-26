package main

import (
	"fmt"
	"github.com/caixr9527/zorm"
	"log"
	"net/http"
)

func Log(next zorm.HandlerFunc) zorm.HandlerFunc {
	return func(ctx *zorm.Context) {
		fmt.Println("pre Log")
		next(ctx)
		fmt.Println("post Log")
	}
}

type User struct {
	Name      string   `xml:"name" json:"name"`
	Age       int      `xml:"age" json:"age" required:"true"`
	Addresses []string `json:"addresses" required:"true"`
}

func main() {
	engine := zorm.New()
	group := engine.Group("user")

	group.Use(func(next zorm.HandlerFunc) zorm.HandlerFunc {
		return func(ctx *zorm.Context) {
			fmt.Println("preHandler")
			next(ctx)
			fmt.Println("postHandler")
		}
	})

	group.Get("/hello", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "hello,world")
	})
	group.Get("/get/:id", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "get user info")
	})
	group.Get("/g/*/get", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "/get/*/get")
	})

	group.Get("/hello/get", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "/hello/get")
	}, Log)

	group.Post("/hello", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "hello,world")
	})
	group.Post("/hello2", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "hello2,world")
	})

	group.Any("/hello3", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "hello3,world")
	})
	group.Get("/html", func(ctx *zorm.Context) {
		ctx.HTML(http.StatusOK, "<h1>hhh</h1>")
	})

	group.Get("/htmlTemplate", func(ctx *zorm.Context) {
		ctx.HTMLTemplate("index.html", "", "tpl/index.html")
	})
	user := &User{Name: "caixiaorong"}
	group.Get("/login", func(ctx *zorm.Context) {
		err := ctx.HTMLTemplate("login.html", user, "tpl/login.html", "tpl/header.html")
		if err != nil {
			log.Println(err)
		}
	})

	group.Get("/htmlTemplateGlob", func(ctx *zorm.Context) {
		err := ctx.HTMLTemplateGlob("login.html", user, "tpl/*.html")
		if err != nil {
			log.Println(err)
		}
	})

	engine.LoadTemplate("tpl/*.html")
	group.Get("template", func(ctx *zorm.Context) {
		user = &User{Name: "caixiaorong1"}
		err := ctx.Template("login.html", user)
		if err != nil {
			log.Println(err)
		}
	})

	group.Get("/json", func(ctx *zorm.Context) {
		user = &User{Name: "caixiaorong1"}
		err := ctx.JSON(http.StatusOK, user)
		if err != nil {
			log.Println(err)
		}
	})

	group.Get("/xml", func(ctx *zorm.Context) {
		user = &User{Name: "caixiaorongxml", Age: 18}
		err := ctx.XML(http.StatusOK, user)
		if err != nil {
			log.Println(err)
		}
	})

	group.Get("/excel", func(ctx *zorm.Context) {
		ctx.File("tpl/1.xlsx")
	})

	group.Get("/excel1", func(ctx *zorm.Context) {
		ctx.FileAttachment("tpl/1.xlsx", "aaaa.xlsx")
	})

	group.Get("/fs", func(ctx *zorm.Context) {
		ctx.FileFromFS("1.xlsx", http.Dir("tpl"))
	})

	group.Get("/redirect", func(ctx *zorm.Context) {
		ctx.Redirect(http.StatusFound, "/user/xml")
	})

	group.Get("/string", func(ctx *zorm.Context) {
		err := ctx.String(http.StatusFound, "和 %s", "你好")
		if err != nil {

		}
	})

	group.Get("/add", func(ctx *zorm.Context) {
		ids, _ := ctx.GetQueryArray("id")
		name := ctx.GetDefaultQuery("name", "zhangsan")
		fmt.Println(ids, name)
	})

	group.Get("/queryMap", func(ctx *zorm.Context) {
		m, _ := ctx.GetQueryMap("user")
		ctx.JSON(http.StatusOK, m)
	})

	group.Post("/formPost", func(ctx *zorm.Context) {
		m, _ := ctx.GetPostFormMap("user")
		//file := ctx.FormFile("file")
		//err := ctx.SaveUploadedFile(file, "./upload/"+file.Filename)
		//if err != nil {
		//	log.Println(err)
		//}
		//form, err := ctx.MultipartForm()
		//if err != nil {
		//	log.Println(err)
		//}
		//fileMap := form.File
		//headers := fileMap["file"]
		headers, _ := ctx.FormFiles("file")
		for _, file := range headers {
			ctx.SaveUploadedFile(file, "./upload/"+file.Filename)
		}

		ctx.JSON(http.StatusOK, m)
	})

	group.Post("/jsonParam", func(ctx *zorm.Context) {
		user := &User{}
		//user := make([]*User, 0)
		ctx.IsValidate = true
		ctx.DisallowUnknownFields = true
		err := ctx.DealJson(user)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}

	})

	engine.Run()
}
