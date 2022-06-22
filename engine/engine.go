package engine

import (
	"errors"
	"github.com/jacoblai/bulu/ketama"
	"github.com/jacoblai/bulu/model"
	"github.com/jacoblai/bulu/rate"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type Engine struct {
	Config      model.Config
	Kts         map[string]*ketama.Continuum
	RateLimiter *rate.RateLimiter
}

func NewEngine(c model.Config) *Engine {
	return &Engine{
		Config: c,
		Kts:    make(map[string]*ketama.Continuum),
	}
}

func (e *Engine) InitNodes(c model.Config) error {
	e.Config = c

	rt, err := time.ParseDuration(e.Config.RateLimit.RateTime)
	if err != nil {
		return err
	}
	e.RateLimiter = rate.NewRateLimiter(rt, e.Config.RateLimit.RateLimit, func() rate.Window {
		return rate.NewLocalWindow()
	})

	for _, v := range e.Config.Domains {
		dm := v.Domain
		nds := make(map[string]uint32)
		for _, n := range v.Nodes {
			nds[n.Url] = n.Weights
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
				log.Println(err)
			} else {
				log.Println("check service alive of", u.Host)
			}
		}

		if len(nds) <= 0 {
			return errors.New("not service alive")
		}

		bks := make([]ketama.Bucket, 0)
		for k, v := range nds {
			bks = append(bks, &model.SimpleBucket{Labels: k, Weights: v})
		}
		e.Kts[dm] = ketama.New(bks)
	}

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
			for _, v := range e.Kts[r.Host].Buckets() {
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
			kt, err := e.GetContinuum(r.Host)
			if err != nil {
				log.Println(err)
			}
			kt.Reset(bks)
			//重定向到活的节点
			node, err := kt.Hash([]byte(r.RemoteAddr))
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
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":false,"errMsg":"Bulu no service alive"}`))
}

func (e *Engine) GetContinuum(host string) (*ketama.Continuum, error) {
	kt, ok := e.Kts[host]
	if !ok {
		kt, ok = e.Kts["*"]
		if !ok {
			return nil, errors.New("not match domain")
		}
	}
	return kt, nil
}
