package main

import (
	"flag"
	"github.com/stvp/aorta/proxy"
	"time"
)

var (
	bind      = flag.String("bind", "0.0.0.0:7979", "bind location for the TCP proxy server")
	password  = flag.String("password", "", "required password before clients can proxy commands")
	clientttl = flag.Int("clientttl", 300, "timeout for client connections, in seconds")
	serverttl = flag.Int("serverttl", 2, "timeout for server connections, in seconds")
)

func main() {
	flag.Parse()

	ctimeout := time.Duration(*clientttl) * time.Second
	stimeout := time.Duration(*serverttl) * time.Second

	server := proxy.NewServer(*bind, *password, ctimeout, stimeout)
	err := server.Listen()
	if err != nil {
		panic(err)
	}
	<-make(chan bool)
}
