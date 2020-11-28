package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/facebookgo/flagenv"
	"github.com/makeworld-the-better-one/go-gemini"
	"github.com/mdlayher/sdnotify"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	certPath = flag.String("cert", "./var/rhea.local.cetacean.club/cert.pem", "TLS certificate path")
	keyPath  = flag.String("key", "./var/rhea.local.cetacean.club/key.pem", "TLS key path")
	port     = flag.Int("port", 1965, "port to listen for Gemini traffic on")
	path     = flag.String("path", "./public", "folder with files to serve")
	httpPort = flag.Int("http-port", 23818, "HTTP server port (for instrumentation, etc)")
)

func main() {
	flagenv.Parse()
	flag.Parse()

	go httpServer()
	go geminiServer()
	n, _ := sdnotify.New()
	n.Notify(sdnotify.Ready)

	log.Printf("listening on gemini=%d, http=%d", *port, *httpPort)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
}

func httpServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), mux)
	if err != nil {
		log.Fatal(err)
	}
}

func geminiServer() {
	err := gemini.ListenAndServe(fmt.Sprintf(":%d", *port), *certPath, *keyPath, server{})
	if err != nil {
		log.Fatal(err)
	}
}

type server struct{}

func (server) Handle(r gemini.Request) *gemini.Response {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "# rhea")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "=> /")

	return &gemini.Response{
		Status: gemini.StatusSuccess,
		Meta:   "text/gemini",
		Body:   ioutil.NopCloser(&buf),
	}
}
