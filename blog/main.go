package main

import (
	"fmt"
	"github.com/caixr9527/zorm"
	"net/http"
)

func main() {
	engine := zorm.New()
	group := engine.Group("user")
	group.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello,world")
	})
	group.Post("/hello2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello2,world")
	})

	group.Any("/hello3", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello3,world")
	})
	engine.Run()
}
