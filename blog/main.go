package main

import "zorm"

func main() {
	engine := zorm.New()
	engine.Run()
}
