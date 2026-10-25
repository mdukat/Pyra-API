package main

import (
	_ "embed"
	"flag"
	"log"
	"net/http"
	"time"
)

//go:embed slownik.md
var dictionaryMarkdown []byte

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	entries := ParseDictionary(dictionaryMarkdown)
	dict := BuildDictionary(entries)
	log.Printf("loaded %d dictionary entries, %d lookup keys", len(entries), len(dict.Index))

	mux := http.NewServeMux()
	mux.HandleFunc("/", infoHandler(dict))
	mux.HandleFunc("/healthz", healthHandler(dict))
	mux.HandleFunc("/translate", translateHandler(dict))
	mux.HandleFunc("/dictionary", dictionaryHandler(dict))

	srv := &http.Server{
		Addr:              *addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("pyra translation API listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
