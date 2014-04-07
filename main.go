package main

import (
	"flag"
	"github.com/stvp/aorta/proxy"
	"github.com/stvp/stvp/log"
	"time"
)

var (
	// Proxy server flags
	bind      = flag.String("bind", "0.0.0.0:7979", "bind location for the TCP proxy server")
	password  = flag.String("password", "", "required password before clients can proxy commands")
	clientttl = flag.Int("clientttl", 300, "timeout for client connections, in seconds")
	serverttl = flag.Int("serverttl", 2, "timeout for server connections, in seconds")

	// Logging flags
	logInterval     = flag.Int("loginterval", 15, "interval, in seconds, to log stats to stdout, LogEntries, etc.")
	env             = flag.String("env", "development", "production, staging, development, test, etc.")
	pagerdutyKey    = flag.String("pagerduty", "", "PagerDuty service key for panic reporting")
	rollbarToken    = flag.String("rollbar", "", "Rollbar token for error logging")
	logentriesToken = flag.String("logentries", "", "Logentries Token for status logging")
	debug           = flag.Bool("debug", false, "log extra debugging info")
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

	go runLogger(server)

	<-make(chan bool)
}

func runLogger(server *proxy.Server) {
	log.Tag = "aorta"
	log.Env = *env
	log.PagerdutyServiceKey = *pagerdutyKey
	log.RollbarToken = *rollbarToken
	log.LogentriesToken = *logentriesToken

	if *debug {
		log.StdoutLevel = log.DEBUG
		log.LogentriesLevel = log.DEBUG
	}

	log.Info("")
	log.Info("              _.---._    /\\\\")
	log.Info("           ./'       \"--`\\//     Aorta " + VERSION)
	log.Info("         ./              o \\")
	log.Info("        /./\\  )______   \\__ \\    Listening: " + *bind)
	log.Info("       ./  / /\\ \\   | \\ \\  \\ \\")
	log.Info("          / /  \\ \\  | |\\ \\  \\7")
	log.Info("           \"     \"    \"  \"")
	log.Info("")

	interval := time.Duration(*logInterval) * time.Second
	for now := range time.Tick(interval) {
		log.Infof("# Stats: %s", now.UTC().Format(time.RFC1123))
		log.Info("")
		log.Info("# Connections")
		log.Infof("current_server_conns:%d", server.Pool.Len())
		log.Infof("current_client_conns:%d", server.CurrentClientConns)
		log.Infof("total_client_conns:%d", server.TotalClientConns)
		log.Info("")
		log.Info("# Cache")
		log.Infof("cache_keys:%d", server.Cache.Len())
		log.Infof("cache_hits:%d", server.Cache.Hits)
		log.Infof("cache_misses:%d", server.Cache.Misses)
		log.Info("")
	}
}
