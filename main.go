package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"

	"github.com/facebookgo/flagenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	err := mime.AddExtensionType(".gmi", "text/gemini")
	if err != nil {
		log.Fatal(err)
	}
	err = mime.AddExtensionType(".gemini", "text/gemini")
	if err != nil {
		log.Fatal(err)
	}
}

var (
	configPath = flag.String("config", "./config.json", "config filename")
)

func main() {
	flagenv.Parse()
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var cfg Config

	fin, err := os.Open(*configPath)
	if err != nil {
		return fmt.Errorf("can't read %s: %v", *configPath, err)
	}
	err = json.NewDecoder(fin).Decode(&cfg)
	if err != nil {
		return fmt.Errorf("can't read %s: %v", *configPath, err)
	}

	go httpServer(cfg)
	go geminiServer(cfg)

	log.Printf("listening on gemini=%d, http=%d", cfg.Port, cfg.HTTPPort)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	fmt.Print("\r")
	log.Println("shutting down")

	return nil
}

func httpServer(cfg Config) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTPPort), mux)
	if err != nil {
		log.Fatal(err)
	}
}

func geminiServer(cfg Config) {
	rh := New(cfg)
	err := rh.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
