package proxy

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/stvp/aorta/proxy"
	"github.com/stvp/tempredis"
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

// -- Helpers

func TestProfile(t *testing.T) {
	// Setup proxy
	p := proxy.NewServer("0.0.0.0:12001", "pw", time.Second, time.Second)
	err := p.Listen()
	if err != nil {
		t.Fatal(err)
	}

	// Setup Redis
	server, err := tempredis.Start(tempredis.Config{"port": "12002", "requirepass": "pw"})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Term()

	// Setup client
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

	// Start profiling
	f, err := os.Create("aorta.prof")
	if err != nil {
		t.Fatal(t)
	}
	pprof.StartCPUProfile(f)
	for i := 0; i < 25000; i++ {
		_, err = client.Do("PING")
		if err != nil {
			t.Fatal(err)
		}
	}
	pprof.StopCPUProfile()

	fmt.Println("Now run:")
	fmt.Println("go tool pprof profile.test aorta.prof")
}
