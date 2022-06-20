package main

import (
	"bulu/ketama"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/libp2p/go-reuseport"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func init() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Fatal(err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Fatal(err)
	}
}

func main() {
	var (
		addr    = flag.String("l", ":7003", "bind host port")
		pemPath = flag.String("pem", "server.pem", "path to pem file")
		keyPath = flag.String("key", "server.key", "path to key file")
		proto   = flag.String("proto", "http", "Proxy protocol (http or https)")
	)
	flag.Parse()

	ks := ketama.New([]ketama.Bucket{
		&SimpleBucket{"http://127.0.0.1:7001/", 100},
		&SimpleBucket{"http://127.0.0.1:7002/", 100},
		&SimpleBucket{"http://127.0.0.1:7004/", 100},
	})

	if *proto != "http" && *proto != "https" {
		log.Fatal("Protocol must be either http or https")
	}

	if *proto == "https" {
		if _, err := os.Stat(*pemPath); err != nil {
			panic(err)
		}
		if _, err := os.Stat(*keyPath); err != nil {
			panic(err)
		}
	}

	srv := &http.Server{
		Addr: *addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			node := ks.Hash([]byte(r.RemoteAddr))
			//log.Printf("proxy_url: %s\n", node.Label())
			u, _ := url.Parse(node.Label())
			proxy := httputil.NewSingleHostReverseProxy(u)
			proxy.ServeHTTP(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	go func() {
		for i := 0; i < runtime.NumCPU(); i++ {
			ln, err := reuseport.Listen("tcp", *addr)
			if err != nil {
				log.Fatal(err)
			}
			if *proto == "http" {
				if err := srv.Serve(ln); err != nil {
					_ = ln.Close()
				}
			} else {
				if err := srv.ServeTLS(ln, *pemPath, *keyPath); err != nil {
					_ = ln.Close()
				}
			}
		}
	}()
	fmt.Printf("Bulu running on port [%s] \n", srv.Addr)

	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	cleanup := make(chan bool)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range signalChan {
			ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)
			go func() {
				_ = srv.Shutdown(ctx)
				cleanup <- true
			}()
			<-cleanup
			fmt.Println("safe exit")
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}

type SimpleBucket struct {
	Labels  string
	Weights uint32
}

func (s *SimpleBucket) Label() string {
	return s.Labels
}

func (s *SimpleBucket) Weight() uint32 {
	return s.Weights
}
