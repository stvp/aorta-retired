package main

import (
	"github.com/stvp/aorta/proxy"
	"time"
)

func main() {
	server := proxy.NewServer("0.0.0.0:9999", "password", time.Minute, time.Second)
	err := server.Listen()
	if err != nil {
		panic(err)
	}
	<-make(chan bool)
}
