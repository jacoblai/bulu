package engine

import (
	"bulu/ketama"
	"bulu/model"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type Engine struct {
	Config model.Config
	Kts    *ketama.Continuum
}

func NewEngine(c model.Config) *Engine {
	return &Engine{
		Config: c,
	}
}

func (e *Engine) Open() error {
	nds := make(map[string]uint32)
	for _, v := range e.Config.Nodes {
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
		bks = append(bks, &model.SimpleBucket{Labels: k, Weights: v})
	}
	e.Kts = ketama.New(bks)

	return nil
}

func (e *Engine) ErrorHandler() func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		//dial tcp 127.0.0.1:7001: connect: connection refused
		if strings.HasSuffix(err.Error(), "connection refused") {
			org := strings.Replace(err.Error(), "dial tcp ", "", 1)
			org = strings.Replace(org, ": connect: connection refused", "", 1)
			//节点列表删除已死节点
			bks := make([]ketama.Bucket, 0)
			for _, v := range e.Kts.Buckets() {
				if strings.Contains(v.Label(), org) {
					continue
				}
				u, _ := url.Parse(v.Label())
				_, err = net.DialTimeout("tcp", u.Host, 2*time.Second)
				if err != nil {
					continue
				}
				bks = append(bks, &model.SimpleBucket{Labels: v.Label(), Weights: v.Weight()})
			}
			if len(bks) <= 0 {
				e.ResultErr(w)
				return
			}
			e.Kts.Reset(bks)
			//重定向到活的节点
			node, err := e.Kts.Hash([]byte(r.RemoteAddr))
			if err != nil {
				e.ResultErr(w)
				return
			}
			//log.Printf("proxy_url_rewrite: %s\n", node.Label())
			u, _ := url.Parse(node.Label())
			proxy := httputil.NewSingleHostReverseProxy(u)
			proxy.ServeHTTP(w, r)
		}
	}
}

func (e *Engine) ResultErr(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Bulu no service"))
}
