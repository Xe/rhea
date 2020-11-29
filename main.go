package main

import (
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"

	"github.com/Xe/rhea/gemini"
	"github.com/facebookgo/flagenv"
	"github.com/mdlayher/sdnotify"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	mime.AddExtensionType("gmi", "text/gemini")
	mime.AddExtensionType("gemini", "text/gemini")
}

var (
	certPath = flag.String("cert", "./var/rhea.local.cetacean.club/cert.pem", "TLS certificate path")
	keyPath  = flag.String("key", "./var/rhea.local.cetacean.club/key.pem", "TLS key path")
	port     = flag.Int("port", 1965, "port to listen for Gemini traffic on")
	path     = flag.String("path", "./public", "folder with files to serve")
	httpPort = flag.Int("http-port", 23818, "HTTP server port (for instrumentation, etc)")
	dbLoc    = flag.String("database-url", "./var/data.db", "SQLite database location")
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
	s := gemini.NewServer(gemini.HandlerFunc(test))
	err := s.ListenAndServe(fmt.Sprintf(":%d", *port), *certPath, *keyPath)
	if err != nil {
		log.Fatal(err)
	}
}

func test(w gemini.ResponseWriter, r *gemini.Request) {
	w.Status(gemini.StatusSuccess, "text/gemini")
	fmt.Fprintln(w, "# hi")
}
