package commonsserver

//
//import (
//	"context"
//	"github.com/stretchr/testify/assert"
//	"net/http"
//	"testing"
//	"time"
//)
//
//func TestServerStartAndShutdown(t *testing.T) {
//	conf := &ServerConf{Port: 80}
//	baseContext, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	handler := http.NewServeMux()
//	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusOK)
//	})
//
//	server := NewServer(conf, baseContext, handler)
//
//	closed := make(chan struct{}, 1)
//
//	go func() {
//		server.Start()
//		closed <- struct{}{}
//	}()
//
//	var resp *http.Response
//	var err error
//	for i := 0; i < 10; i++ {
//		resp, err = http.Get("http://localhost:80")
//		if err == nil {
//			break
//		}
//		time.Sleep(100 * time.Millisecond)
//	}
//
//	if err != nil {
//		t.Fatalf("server did not start: %v", err)
//	}
//
//	assert.Nil(t, err)
//	defer resp.Body.Close()
//	assert.Equal(t, http.StatusOK, resp.StatusCode)
//
//	cancel()
//
//	select {
//	case <-closed:
//	case <-time.After(2 * time.Second):
//		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
//		defer cancel()
//		if err := server.httpServer.Shutdown(c); err != nil {
//			t.Fatalf("server did not shutdown: %v", err)
//		}
//		t.Fatal("server did not shut down gracefully")
//	}
//}
