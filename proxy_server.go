package main

import (
	"github.com/stvp/resp"
	"net"
	"strconv"
	"strings"
	"time"
)

var ()

type ProxyServer struct {
	// Settings
	password      string
	clientTimeout time.Duration
	serverTimeout time.Duration

	bind string
	pool *ServerConnPool
	// cache *Cache
}

func NewProxyServer(bind, password string, clientTimeout, serverTimeout time.Duration) *ProxyServer {
	return &ProxyServer{
		password:      password,
		clientTimeout: clientTimeout,
		serverTimeout: serverTimeout,
		bind:          bind,
		pool:          NewServerConnPool(),
	}
}

// TODO trap TERM and give goroutines a second or two to finish up
func (s *ProxyServer) Run() error {
	listener, err := net.Listen("tcp", s.bind)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handle(conn)
	}

	return nil
}

func (s *ProxyServer) handle(conn net.Conn) {
	client := NewClientConn(conn, s.clientTimeout)
	defer client.Close()
	var server *ServerConn
	var authenticated bool

	for {
		// Read command
		command, err := client.ReadCommand()
		if err == ErrTimeout || err == ErrConnClosed {
			return
		} else if err == resp.ErrSyntaxError {
			client.WriteError("ERR syntax error")
			return
		} else if err != nil {
			client.WriteError("aorta: " + err.Error())
			return
		}

		// Parse command
		args, err := command.Strings()
		if err != nil {
			client.WriteError("ERR syntax error")
		}
		commandName := strings.ToUpper(args[0])

		// Handle authentication
		if commandName == "AUTH" {
			if len(args) != 2 {
				client.WriteError("ERR wrong number of arguments for 'auth' command")
			} else if args[1] == s.password {
				authenticated = true
				client.Write(resp.OK)
			} else {
				authenticated = false
				client.WriteError("ERR invalid password")
			}
			continue
		}
		if !authenticated {
			client.WriteError("NOAUTH Authentication required.")
			return
		}

		// Handle server destination
		if commandName == "PROXY" {
			if len(args) != 4 {
				client.WriteError("ERR wrong number of arguments for 'proxy' command")
			}
			server = s.pool.Get(args[1], args[2], args[3])
			client.Write(resp.OK)
			continue
		}
		if server == nil {
			client.WriteError("aorta: proxy destination not set")
		}

		// Handle the command
		switch commandName {
		case "CACHED":
			if len(args) < 3 {
				client.WriteError("ERR wrong number of arguments for 'cached' command")
				return
			}
			secs, err := strconv.Atoi(args[1])
			if err != nil {
				client.WriteError("ERR syntax error")
			}
			panic(secs)
			// TODO cached get
		case "QUIT":
			return
		default:
			// response, err := server.Do(command)
			// if err != nil {
			// // TODO figure out what errors we might see here
			// client.WriteError(err.Error())
			// continue
			// }

			// err = client.Write(response)
			// if err != nil {
			// return
			// }
		}
	}
}
