package main

import (
	"time"
)

func main() {
	server := NewProxyServer("0.0.0.0:9999", "password", time.Minute, time.Second)
	err := server.Run()
	if err != nil {
		panic(err)
	}
}
