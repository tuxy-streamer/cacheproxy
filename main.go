package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type Cache struct {
	response     *http.Response
	responseBody []byte
	createdAt    time.Time
}

type Proxy struct {
	originURL string
	Cache     map[string]*Cache
	mutex     sync.RWMutex
}

func NewProxy(originURL string) *Proxy {
	return &Proxy{
		originURL: originURL,
		Cache:     make(map[string]*Cache),
	}
}

func (p *Proxy) clearCache() {
	p.Cache = make(map[string]*Cache)
	fmt.Println("Info: cache cleaned successfully.")
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	cacheKey := req.URL.String() + " : " + req.Method
	cacheStatus := "MISS"
	c, ok := p.Cache[cacheKey]
	if ok {
		cacheStatus = "HIT"
	} else {
		cacheStatus = "MISS"
		url := p.originURL + req.URL.String()
		res, err := http.Get(url)
		if err != nil {
			http.Error(rw, "Error: forwarding request: ", http.StatusInternalServerError)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			http.Error(rw, "Error Forwarding Request Body", http.StatusInternalServerError)
			return
		}
		p.Cache[cacheKey] = &Cache{response: res, responseBody: body, createdAt: time.Now()}
		c = p.Cache[cacheKey]
	}

	for key, values := range c.response.Header {
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}
	rw.Header().Set("X-Cache", cacheStatus)
	rw.WriteHeader(c.response.StatusCode)
	rw.Write(c.responseBody)
}

func main() {
	portFlag := flag.String("port", "3000", "Proxy Port")
	originFlag := flag.String("origin", "http://dummyjson.com", "Origin URL")
	clearCacheFlag := flag.Bool("clear-cache", false, "Clear Cache")
	flag.Parse()
	proxy := NewProxy(*originFlag)
	if *clearCacheFlag {
		proxy.clearCache()
		os.Exit(0)
	}
	err := http.ListenAndServe(":"+*portFlag, proxy)
	if err != nil {
		fmt.Println("Fatal: Failed to start server.")
	}
}
