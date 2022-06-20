package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/libp2p/go-reuseport"
	"io"
	"log"
	"net"
	"net/http"
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
		addr    = flag.String("l", ":7003", "绑定Host地址")
		pemPath = flag.String("pem", "server.pem", "path to pem file")
		keyPath = flag.String("key", "server.key", "path to key file")
		proto   = flag.String("proto", "http", "Proxy protocol (http or https)")
	)
	flag.Parse()
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
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleHTTP(w, r)
			}
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

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	_, _ = io.Copy(destination, source)
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
