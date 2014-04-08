package main

import (
	"flag"
	"github.com/stvp/aorta/proxy"
	"github.com/stvp/stvp/log"
	. "github.com/stvp/stvp/log/helpers"
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

func init() {
	flag.Parse()

	log.Tag = "aorta"
	log.Env = *env
	log.PagerdutyServiceKey = *pagerdutyKey
	log.RollbarToken = *rollbarToken
	log.LogentriesToken = *logentriesToken

	if *debug {
		log.StdoutLevel = log.DEBUG
		log.LogentriesLevel = log.DEBUG
	}
}

func main() {
	// Trigger PagerDuty if we panic
	defer log.LogPanic(log.CRIT)

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
	INFO("")
	INFO("              _.---._    /\\\\")
	INFO("           ./'       \"--`\\//     Aorta %s", VERSION)
	INFO("         ./              o \\")
	INFO("        /./\\  )______   \\__ \\    Listening: %s", *bind)
	INFO("       ./  / /\\ \\   | \\ \\  \\ \\   Env: %s", *env)
	INFO("          / /  \\ \\  | |\\ \\  \\7")
	INFO("           \"     \"    \"  \"")
	INFO("")

	interval := time.Duration(*logInterval) * time.Second
	for now := range time.Tick(interval) {
		INFO("# Stats @ %s", now.UTC().Format(time.RFC1123))
		INFO("current_server_conns:%d\tcurrent_client_conns:%d\ttotal_client_conns:%d", server.Pool.Len(), server.CurrentClientConns, server.TotalClientConns)
		INFO("cache_keys:%d\tcache_hits:%d\tcache_misses:%d", server.Cache.Len(), server.Cache.Hits, server.Cache.Misses)
	}
}
