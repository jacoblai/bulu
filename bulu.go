package main

import (
	"bulu/ketama"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-reuseport"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
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

type Config struct {
	Host    string `json:"host"`
	PemPath string `json:"pemPath"`
	KeyPath string `json:"keyPath"`
	Proto   string `json:"proto"`
	Nodes   []Node `json:"nodes"`
}

type Node struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Weights uint32 `json:"weights"`
}

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	confPath := dir + "/bulu_conf.js"
	if _, err := os.Stat(confPath); err != nil {
		log.Fatal("bulu_conf config file not found..")
	}

	bts, err := os.ReadFile(confPath)
	if err != nil {
		log.Fatal(err)
	}

	var conf Config
	err = json.Unmarshal(bts, &conf)
	if err != nil {
		log.Fatal(err)
	}

	if conf.Proto != "http" && conf.Proto != "https" {
		log.Fatal("Protocol must be either http or https")
	}

	nds := make(map[string]uint32)
	for _, v := range conf.Nodes {
		nds[v.Url] = v.Weights
	}
	for k := range nds {
		u, err := url.Parse(k)
		if err != nil {
			delete(nds, k)
			log.Println(err)
			continue
		}
		_, err = net.DialTimeout("tcp", u.Host, 2*time.Second)
		if err != nil {
			delete(nds, k)
			log.Println("Site unreachable", err)
		} else {
			log.Println("check service alive of", u.Host)
		}
	}

	bks := make([]ketama.Bucket, 0)
	for k, v := range nds {
		bks = append(bks, &SimpleBucket{k, v})
	}
	ks := ketama.New(bks)

	if conf.Proto == "https" {
		if _, err := os.Stat(conf.PemPath); err != nil {
			panic(err)
		}
		if _, err := os.Stat(conf.KeyPath); err != nil {
			panic(err)
		}
	}

	srv := &http.Server{
		Addr: conf.Host,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			node := ks.Hash([]byte(r.RemoteAddr))
			log.Printf("proxy_url: %s\n", node.Label())
			u, _ := url.Parse(node.Label())
			proxy := httputil.NewSingleHostReverseProxy(u)
			proxy.ErrorHandler = errorHandler()
			proxy.ServeHTTP(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	go func() {
		for i := 0; i < runtime.NumCPU(); i++ {
			ln, err := reuseport.Listen("tcp", conf.Host)
			if err != nil {
				log.Fatal(err)
			}
			if conf.Proto == "http" {
				if err := srv.Serve(ln); err != nil {
					_ = ln.Close()
				}
			} else {
				if err := srv.ServeTLS(ln, conf.PemPath, conf.KeyPath); err != nil {
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

func errorHandler() func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, req *http.Request, err error) {
		fmt.Printf("Got error while modifying response: %v \n", err)
		return
	}
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
