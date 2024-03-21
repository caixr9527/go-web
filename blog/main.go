package main

import (
	"fmt"
	"github.com/caixr9527/zorm"
)

func main() {
	engine := zorm.New()
	group := engine.Group("user")
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
	})

	group.Post("/hello", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "hello,world")
	})
	group.Post("/hello2", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "hello2,world")
	})

	group.Any("/hello3", func(ctx *zorm.Context) {
		fmt.Fprintf(ctx.W, "hello3,world")
	})
	engine.Run()
}
