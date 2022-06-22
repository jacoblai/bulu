package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jacoblai/bulu/engine"
	"github.com/jacoblai/bulu/model"
	"github.com/libp2p/go-reuseport"
	"log"
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

func main() {
	var (
		confPath = flag.String("c", ".", "config file path")
	)
	flag.Parse()
	if *confPath == "." {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}
		*confPath = dir + "/bulu_conf.js"
	}

	if _, err := os.Stat(*confPath); err != nil {
		log.Fatal("bulu config file not found..")
	}

	bts, err := os.ReadFile(*confPath)
	if err != nil {
		log.Fatal("read config file error..")
	}

	var conf model.Config
	err = json.Unmarshal(bts, &conf)
	if err != nil {
		log.Fatal("config file content error..")
	}

	if conf.Proto != "http" && conf.Proto != "https" {
		log.Fatal("Protocol must be either http or https")
	}

	eng := engine.NewEngine(conf)
	err = eng.InitNodes(conf)
	if err != nil {
		log.Fatal(err)
	}
	eng.Watcher(*confPath)

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
		Handler: eng.JwtAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			node, err := eng.Kts.Hash([]byte(r.RemoteAddr))
			if err != nil {
				eng.ResultErr(w)
				return
			}
			//log.Printf("proxy_url: %s\n", node.Label())
			u, _ := url.Parse(node.Label())
			proxy := httputil.NewSingleHostReverseProxy(u)
			proxy.ErrorHandler = eng.ErrorHandler()
			proxy.ServeHTTP(w, r)
		})),
		// Disable HTTP/2. 防止却持
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
