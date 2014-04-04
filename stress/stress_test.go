package proxy

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/stvp/aorta/proxy"
	"github.com/stvp/tempredis"
	"strconv"
	"testing"
	"time"
)

func TestStress(t *testing.T) {
	serverCount := 4
	goroutineCount := 4

	p := proxy.NewServer("0.0.0.0:12001", "pw", time.Second, time.Second)
	err := p.Listen()
	if err != nil {
		panic(err)
	}

	servers := make([]*tempredis.Server, serverCount)
	for i := 0; i < serverCount; i++ {
		server, err := tempredis.Start(tempredis.Config{
			"port":        strconv.Itoa(22000 + i),
			"requirepass": strconv.Itoa(i),
		})
		if err != nil {
			t.Fatal(err)
		}
		defer server.Term()

		client, err := redis.Dial("tcp", "0.0.0.0:12001")
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Do("AUTH", "pw")
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Do("PROXY", server.Config.Bind(), server.Config.Port(), server.Config.Password())
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Do("SET", "i", i)
		if err != nil {
			t.Fatal(err)
		}

		servers[i] = server
	}

	for g := 0; g < goroutineCount; g++ {
		gnum := g
		go func() {
			client, err := redis.Dial("tcp", "0.0.0.0:12001")
			if err != nil {
				panic(err)
			}
			_, err = client.Do("AUTH", "pw")
			if err != nil {
				panic(err)
			}
			var counter int64
			for {
				i := counter % int64(len(servers))
				_, err = client.Do("PROXY", servers[i].Config.Bind(), servers[i].Config.Port(), servers[i].Config.Password())
				if err != nil {
					panic(err)
				}

				got, err := redis.Int64(client.Do("GET", "i"))
				if err != nil {
					panic(err)
				}
				if got != i {
					fmt.Printf("got %d, expected %d\n", got, i)
					panic(err)
				}

				if counter%10000 == 0 {
					fmt.Printf("goroutine %d: %d queries ok\n", gnum, counter)
				}
				counter++
			}
		}()
	}

	<-make(chan bool)
}
