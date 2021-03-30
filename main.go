package main

import (
	"context"
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
	"within.website/ln"
)

func init() {
	ctx := context.Background()
	err := mime.AddExtensionType(".gmi", "text/gemini")
	if err != nil {
		ln.FatalErr(ctx, err)
	}
	err = mime.AddExtensionType(".gemini", "text/gemini")
	if err != nil {
		ln.FatalErr(ctx, err)
	}
}

var (
	configPath = flag.String("config", "./config.json", "config filename")
)

func main() {
	flagenv.Parse()
	flag.Parse()

	ctx := context.Background()

	err := run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	var cfg Config

	fin, err := os.Open(*configPath)
	if err != nil {
		return fmt.Errorf("can't read %s: %v", *configPath, err)
	}
	err = json.NewDecoder(fin).Decode(&cfg)
	if err != nil {
		return fmt.Errorf("can't read %s: %v", *configPath, err)
	}

	go httpServer(ctx, cfg)
	go geminiServer(ctx, cfg)

	for _, site := range cfg.Sites {
		ln.Log(ctx, ln.Info("loaded site %s", site.Domain))
	}
	ln.Log(ctx, ln.Info("listening on gemini=%d http=%d", cfg.Port, cfg.HTTPPort))

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	fmt.Print("\r")
	ln.Log(ctx, ln.Info("shutting down"))

	return nil
}

func httpServer(ctx context.Context, cfg Config) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTPPort), mux)
	if err != nil {
		ln.FatalErr(ctx, err)
	}
}

func geminiServer(ctx context.Context, cfg Config) {
	rh := New(cfg)
	err := rh.ListenAndServe()
	if err != nil {
		ln.FatalErr(ctx, err)
	}
}
