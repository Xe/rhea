package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Xe/rhea/gemini"
	"github.com/mdlayher/sdnotify"
)

type Rhea struct {
	cfg Config
}

func New(cfg Config) Rhea {
	return Rhea{cfg: cfg}
}

func (rh *Rhea) tlsConfig() *tls.Config {
	result := &tls.Config{}

	for _, site := range rh.cfg.Sites {
		cert, err := tls.LoadX509KeyPair(site.CertPath, site.KeyPath)
		if err != nil {
			log.Panicf("error loading certs for %s: %v", site.Domain, err)
		}
		result.Certificates = append(result.Certificates, cert)
	}

	return result
}

func (rh *Rhea) ListenAndServe() error {
	lis, err := tls.Listen("tcp", fmt.Sprintf(":%d", rh.cfg.Port), rh.tlsConfig())
	if err != nil {
		return fmt.Errorf("can't listen on port %d: %v", rh.cfg.Port, err)
	}
	s := gemini.NewServer(rh)

	n, _ := sdnotify.New()
	n.Notify(sdnotify.Ready)
	n.Notify(sdnotify.Statusf("serving %d sites", len(rh.cfg.Sites)))
	return s.Serve(lis)
}

func (rh *Rhea) HandleGemini(w gemini.ResponseWriter, r *gemini.Request) {
	host := r.URL.Host

	if strings.Contains(host, ":") {
		rawHost, _, err := net.SplitHostPort(r.URL.Host)
		if err != nil {
			w.Status(gemini.StatusBadRequest, "unparseable url host")
			return
		}

		host = rawHost
	}

	for _, s := range rh.cfg.Sites {
		if host == s.Domain {
			s.HandleGemini(w, r)
			return
		}
	}

	w.Status(gemini.StatusProxyRequestRefused, fmt.Sprintf("can't proxy to %s", host))
}

func (s Site) HandleGemini(w gemini.ResponseWriter, r *gemini.Request) {
	if s.Files != nil {
		s.Files.HandleGemini(w, r)
		return
	}

	w.Status(gemini.StatusUnavailable, "no active configuration detected")
}

func (f FileServer) writeIndex(path string, r *gemini.Request, w gemini.ResponseWriter) {
	dir, err := os.Open(path)
	if err != nil {
		w.Status(gemini.StatusNotFound, err.Error())
		return
	}

	names, err := dir.Readdirnames(0)
	if err != nil {
		w.Status(gemini.StatusPermanentFailure, err.Error())
		return
	}

	sort.Strings(names)

	w.Status(gemini.StatusSuccess, "text/gemini")

	fmt.Fprintf(w, "# Index for %s\n", r.URL.Path)
	fmt.Fprintln(w, "=> .. ..")

	for _, name := range names {
		fmt.Fprintf(w, "=> %[1]s %[1]s", name)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Served by rhea")
}

func (f FileServer) HandleGemini(w gemini.ResponseWriter, r *gemini.Request) {
	path := filepath.Join(f.Root, r.URL.Path)
	st, err := os.Stat(path)
	if err != nil {
		w.Status(gemini.StatusNotFound, fmt.Sprint("can't find ", r.URL.Path))
		log.Printf("%v", err)
		return
	}

	if st.IsDir() {
		newPath := filepath.Join(path, "index.gmi")
		_, err := os.Stat(newPath)
		if err != nil {
			if f.AutoIndex {
				f.writeIndex(path, r, w)
				return
			}
			w.Status(gemini.StatusNotFound, "this is a folder, but has no index")
			return
		}
		path = newPath
	}

	fin, err := os.Open(path)
	if err != nil {
		w.Status(gemini.StatusTemporaryFailure, "can't open file")
		log.Printf("%v", err)
		return
	}
	defer fin.Close()

	mimeT := mime.TypeByExtension(filepath.Ext(path))
	w.Status(gemini.StatusSuccess, mimeT)
	io.Copy(w, fin)
}
