package main

import (
	"github.com/stvp/resp"
	"net"
	"strconv"
	"strings"
	"time"
)

type ProxyServer struct {
	// Settings
	password      string
	clientTimeout time.Duration
	serverTimeout time.Duration

	bind     string
	listener net.Listener
	pool     *ServerConnPool
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

func (s *ProxyServer) Listen() error {
	listener, err := net.Listen("tcp", s.bind)
	if err != nil {
		return err
	}
	s.listener = listener

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// TODO surface error to logger or something. Also, closing the
				// listener returns an error here, so we should ignore that.
				return
			}
			go s.handle(conn)
		}
	}()

	return nil
}

func (s *ProxyServer) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
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
			// Redis returns the period even though thats inconsistent with all other
			// error messages. We include it here for correctness.
			client.WriteError("NOAUTH Authentication required.")
			return
		}

		// Handle server destination
		if commandName == "PROXY" {
			if len(args) != 4 {
				client.WriteError("ERR wrong number of arguments for 'proxy' command")
				continue
			}
			server = s.pool.Get(args[1], args[2], args[3])
			client.Write(resp.OK)
			continue
		}

		if server == nil {
			client.WriteError("aorta: proxy destination not set")
			continue
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
			response, err := server.Do(command)
			if err != nil {
				switch e := err.(type) {
				case resp.Error:
					client.Write(e)
				default:
					client.WriteError(err.Error())
				}
				continue
			}

			err = client.Write(response.([]byte))
			if err != nil {
				return
			}
		}
	}
}
