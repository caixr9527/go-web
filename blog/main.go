package main

import (
	"errors"
	"fmt"
	"github.com/caixr9527/zorm"
	zormlog "github.com/caixr9527/zorm/log"
	"github.com/caixr9527/zorm/token"
	"github.com/caixr9527/zorm/zerror"
	"github.com/caixr9527/zorm/zpool"
	"log"
	"net/http"
	"sync"
	"time"
)

func Log(next zorm.HandlerFunc) zorm.HandlerFunc {
	return func(ctx *zorm.Context) {
		fmt.Println("pre Log")
		next(ctx)
		fmt.Println("post Log")
	}
}

type User struct {
	Name      string   `xml:"name" json:"name" `
	Age       int      `xml:"age" json:"age" required:"true" validate:"required,max=50,min=18"`
	Addresses []string `json:"addresses" required:"true"`
}

func main() {
	engine := zorm.Default()
	engine.RegisterErrorHandler(func(err error) (int, any) {
		switch e := err.(type) {
		case *BlogResponse:
			return http.StatusOK, e.Response()
		default:
			return http.StatusInternalServerError, "500 error"
		}
	})
	//fmt.Println(zorm.BasicAuth("caixr", "123456"))
	//auth := &zorm.Accounts{
	//	Users: make(map[string]string),
	//}
	//auth.Users["caixr"] = "123456"
	//engine.Use(auth.BasicAuth)
	jh := &token.JwtHandler{
		Key: []byte("123456"),
	}
	engine.Use(jh.AuthInterceptor)
	group := engine.Group("user")
	group.Use(zorm.Logging, zorm.Recovery)

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

	//logger := zormlog.Default()
	logger := engine.Logger
	logger.Level = zormlog.Debug
	//logger.Formatter = &zormlog.JsonFormatter{TimeDisplay: true}
	//logger.Outs = append(logger.Outs, zormlog.FileWrite("./log/log.log"))
	//logger.SetLogPath("./log")
	//fmt.Println(config.Conf)
	//logger.LogFileSize = 1 << 10
	//var u *User
	group.Post("/jsonParam", func(ctx *zorm.Context) {
		//user := &User{}
		user := make([]User, 0)
		//u.Age = 10
		ctx.IsValidate = true
		ctx.DisallowUnknownFields = true
		err := ctx.BindJson(&user)
		ctx.Logger.WithFields(zormlog.Fields{
			"name": "caixiaorong",
			"id":   1000,
		}).Info("info")
		ctx.Logger.Debug("debug")
		ctx.Logger.Error("error")

		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}

	})

	group.Post("/xmlParam", func(ctx *zorm.Context) {
		user := &User{}
		err := ctx.BindXML(user)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}

	})

	group.Get("/errorTest", func(ctx *zorm.Context) {
		//zError := zerror.Default()
		//zError.Result(func(zError *zerror.ZError) {
		//	ctx.Logger.Error(zError.Error())
		//	ctx.JSON(http.StatusInternalServerError, nil)
		//})
		//a(1, zError)
		//b(1, zError)
		//c(1, zError)
		err := login()
		ctx.HandlerWithError(http.StatusOK, user, err)
	})

	pool, _ := zpool.NewPool(5)
	group.Get("/pool", func(ctx *zorm.Context) {
		now := time.Now()
		var wg sync.WaitGroup
		wg.Add(5)
		pool.Submit(func() {
			fmt.Println("1111111")
			wg.Done()
			panic("panic")
			time.Sleep(3 * time.Second)
		})
		pool.Submit(func() {
			fmt.Println("2222222")
			time.Sleep(3 * time.Second)
			wg.Done()
		})
		pool.Submit(func() {
			fmt.Println("3333333")
			time.Sleep(3 * time.Second)
			wg.Done()
		})
		pool.Submit(func() {
			fmt.Println("4444444")
			time.Sleep(3 * time.Second)
			wg.Done()
		})
		pool.Submit(func() {
			fmt.Println("5555555")
			time.Sleep(3 * time.Second)
			wg.Done()
		})
		wg.Wait()
		fmt.Printf("time: %v\n", time.Now().UnixMilli()-now.UnixMilli())
		ctx.JSON(http.StatusOK, "success")
	})
	group.Get("/loginToken", func(ctx *zorm.Context) {
		jwt := &token.JwtHandler{}
		jwt.Key = []byte("123456")
		jwt.SendCookie = true
		jwt.TimeOut = 10 * time.Minute
		jwt.RefreshTimeOut = 2 * 60 * time.Minute
		jwt.Authenticator = func(ctx *zorm.Context) (map[string]any, error) {
			data := make(map[string]any)
			data["userId"] = 1
			return data, nil
		}
		jwtResponse, err := jwt.LoginHandler(ctx)
		if err != nil {
			log.Println(err)
			ctx.JSON(http.StatusOK, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, jwtResponse)
	})

	group.Get("/refreshToken", func(ctx *zorm.Context) {
		jwt := &token.JwtHandler{}
		jwt.Key = []byte("12346")
		jwt.SendCookie = true
		jwt.TimeOut = 60 * time.Second
		jwt.RefreshTimeOut = 2 * 60 * time.Second
		jwt.RefreshKey = "blog_refresh_token"
		ctx.Set(jwt.RefreshKey, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTI2ODAyODYsImlhdCI6MTcxMjY3MzA4NiwidXNlcklkIjoxfQ.LbSM0lsnIBWSx40nV-OHsu-ebPVYpg2msKEkYaqhPVc")
		jwt.Authenticator = func(ctx *zorm.Context) (map[string]any, error) {
			data := make(map[string]any)
			data["userId"] = 1
			return data, nil
		}
		jwtResponse, err := jwt.RefreshHandler(ctx)
		if err != nil {
			log.Println(err)
			ctx.JSON(http.StatusOK, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, jwtResponse)
	})
	//engine.Run()
	engine.RunTLS(":8118", "key/server.pem", "key/server.key")
}

type BlogResponse struct {
	Success bool
	Code    int
	Data    any
	Msg     string
}

func (b *BlogResponse) Error() string {
	return b.Msg
}

func (b *BlogResponse) Response() any {
	return b
}

func login() *BlogResponse {
	return &BlogResponse{
		Success: false,
		Code:    -999,
		Data:    nil,
		Msg:     "login error"}
}

func a(i int, zError *zerror.ZError) {
	if i == 1 {
		err := errors.New("a error")
		zError.Put(err)
	}

}

func b(i int, zError *zerror.ZError) {
	if i == 1 {
		err := errors.New("a error")
		zError.Put(err)
	}

}

func c(i int, zError *zerror.ZError) {
	if i == 1 {
		err := errors.New("a error")
		zError.Put(err)
	}

}
