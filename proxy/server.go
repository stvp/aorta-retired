package proxy

import (
	"bytes"
	"fmt"
	"github.com/stvp/aorta/cache"
	"github.com/stvp/aorta/redis"
	"github.com/stvp/resp"
	. "github.com/stvp/stvp/log/helpers"
	"net"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	// Settings
	password      string
	clientTimeout time.Duration
	serverTimeout time.Duration

	bind     string
	listener net.Listener
	Pool     *redis.ServerConnPool
	Cache    *cache.Cache

	// Stats
	TotalClientConns   int
	CurrentClientConns int
}

func NewServer(bind, password string, clientTimeout, serverTimeout time.Duration) *Server {
	return &Server{
		password:      password,
		clientTimeout: clientTimeout,
		serverTimeout: serverTimeout,
		bind:          bind,
		Pool:          redis.NewServerConnPool(),
		Cache:         cache.NewCache(),
	}
}

func (s *Server) Listen() error {
	listener, err := net.Listen("tcp", s.bind)
	if err != nil {
		return err
	}
	s.listener = listener

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				ERROR(err.Error())
				return
			}
			DEBUG("New client: %s", conn.RemoteAddr().String())
			go s.handle(conn)
		}
	}()

	return nil
}

func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) handle(conn net.Conn) {
	client := redis.NewClientConn(conn, s.clientTimeout)
	defer client.Close()

	s.TotalClientConns++
	s.CurrentClientConns++
	defer func() {
		s.CurrentClientConns--
		DEBUG("Closed client: %s", conn.RemoteAddr().String())
	}()

	// State
	var authenticated bool
	var server *redis.ServerConn

	for {
		// Read command
		command, err := client.ReadCommand()
		if err == redis.ErrTimeout || err == redis.ErrConnClosed {
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
			return
		}
		commandName := strings.ToUpper(args[0])

		if commandName == "QUIT" {
			return
		}

		// Require authentication
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

		// Require destination server
		if commandName == "PROXY" {
			server = nil
			if len(args) != 4 {
				client.WriteError("ERR wrong number of arguments for 'proxy' command")
				continue
			}
			address := fmt.Sprintf("%s:%s", args[1], args[2])
			server = s.Pool.Get(address, args[3], s.serverTimeout)
			client.Write(resp.OK)
			continue
		}

		if server == nil {
			client.WriteError("aorta: proxy destination not set")
			continue
		}

		// Handle CACHED command prefix
		var maxAge time.Time
		if commandName == "CACHED" {
			if len(args) < 3 {
				client.WriteError("ERR wrong number of arguments for 'cached' command")
				return
			}
			secs, err := strconv.Atoi(args[1])
			if err != nil {
				client.WriteError("ERR syntax error")
			}
			maxAge = time.Now().Add(-time.Duration(secs) * time.Second)
			command = resp.NewCommand((args[2:])...)
		} else {
			maxAge = time.Now()
		}

		// Handle the command
		response, err := s.cachedDo(maxAge, command, server)
		if err != nil {
			client.WriteError(err.Error())
			continue
		}

		err = client.Write(response.Raw())
		if err != nil {
			return
		}
	}
}

func (s *Server) cachedDo(maxAge time.Time, command resp.Command, conn *redis.ServerConn) (resp.Object, error) {
	key := s.cacheKey(command, conn)
	return s.Cache.Fetch(key, maxAge, func() (resp.Object, error) {
		return conn.Do(command)
	})
}

func (s *Server) cacheKey(command resp.Command, conn *redis.ServerConn) string {
	var buf bytes.Buffer
	buf.WriteString(conn.Address())
	buf.WriteString(conn.Password())
	args, _ := command.Slices()
	for _, arg := range args {
		buf.Write(arg)
	}
	return buf.String()
}
