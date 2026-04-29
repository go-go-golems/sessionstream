package main

import (
	"flag"
	"log"
	"net/http"
	"time"
)

func main() {
	addr := flag.String("addr", ":8091", "listen address")
	flag.Parse()

	app, err := newSystemlabServer()
	if err != nil {
		log.Fatalf("build systemlab server: %v", err)
	}
	log.Printf("sessionstream-systemlab listening on %s", *addr)
	server := &http.Server{
		Addr:              *addr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}
