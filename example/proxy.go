package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	bwlimit "github.com/u-haru/go-bwlimit"
)

var (
	localHost string
	speed     uint64
)

func HandleHttps(writer http.ResponseWriter, req *http.Request) {
	destConn, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		log.Print(err)
		return
	}
	writer.WriteHeader(200)

	if clientConn, _, err := writer.(http.Hijacker).Hijack(); err != nil {
		log.Print(err)
		return
	} else {
		go func() {
			bwlimit.Copy(clientConn, destConn, speed) // here
			destConn.Close()
		}()
		go func() {
			bwlimit.Copy(destConn, clientConn, speed) // here
			clientConn.Close()
		}()
	}
}

func HandleHttp(writer http.ResponseWriter, req *http.Request) {
	req.RequestURI = ""
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Print(err)
		return
	}
	for v, k := range resp.Header {
		writer.Header()[v] = k
	}
	bwlimit.Copy(writer, resp.Body, speed) // here
	resp.Body.Close()
}

func HandleRequest(writer http.ResponseWriter, req *http.Request) {
	if req.Method == "CONNECT" {
		HandleHttps(writer, req)
	} else {
		HandleHttp(writer, req)
	}
}

func main() {
	flag.StringVar(&localHost, "l", "0.0.0.0:8080", "Address:port")
	flag.Uint64Var(&speed, "s", 124*bwlimit.KB, "Bps")
	flag.BoolVar(&bwlimit.Debug, "d", false, "debug")
	flag.Parse()

	srv := &http.Server{
		Addr:    localHost,
		Handler: http.HandlerFunc(HandleRequest),
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("Server closed with error:", err)
		}
	}()
	log.Printf("Start serving on %s", localHost)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
	log.Printf("SIGNAL %d received, shutting down...\n", <-quit)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("Failed to gracefully shutdown:", err)
	}
	log.Println("Server shutdown")
}
