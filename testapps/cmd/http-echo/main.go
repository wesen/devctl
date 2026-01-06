package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 0, "Port to listen on (0 for ephemeral)")
	flag.Parse()

	if port == 0 {
		if v := os.Getenv("HTTP_ECHO_PORT"); v != "" {
			_, _ = fmt.Sscanf(v, "%d", &port)
		}
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "listen error: %v\n", err)
		os.Exit(2)
	}
	actualAddr := ln.Addr().String()
	_, _ = fmt.Fprintf(os.Stderr, "listening on %s\n", actualAddr)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()
		for range t.C {
			_, _ = fmt.Fprintln(os.Stdout, "http-echo: tick")
		}
	}()

	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		_, _ = fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
		os.Exit(3)
	}
}
