package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func main() {
	target := "http://localhost:4444"

	targetURL, err := url.Parse(target)
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.Host = targetURL.Host
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Content-Encoding")
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Printf("ERROR: %s %s - %v", req.Method, req.URL.Path, err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Proxy error: " + err.Error()))
	}

	log.Println("Starting FE proxy on http://localhost:4731 -> http://localhost:4444")

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		log.Printf("-> %s %s", req.Method, req.URL.Path)
		proxy.ServeHTTP(w, req)
		log.Printf("<- %s %s %d %v", req.Method, req.URL.Path, 200, time.Since(start))
	})

	if err := http.ListenAndServe(":4731", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
