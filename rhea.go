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
	"strings"

	"github.com/Xe/rhea/gemini"
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

func (f FileServer) HandleGemini(w gemini.ResponseWriter, r *gemini.Request) {
	path := filepath.Join(f.Root, r.URL.Path)
	st, err := os.Stat(path)
	if err != nil {
		w.Status(gemini.StatusNotFound, fmt.Sprint("can't find", r.URL.Path))
		return
	}

	if st.IsDir() {
		newPath := filepath.Join(path, "index.gmi")
		_, err := os.Stat(newPath)
		if err != nil {
			w.Status(gemini.StatusNotFound, "this is a folder, but has no index")
			return
		}
		path = newPath
	}

	fin, err := os.Open(path)
	if err != nil {
		w.Status(gemini.StatusTemporaryFailure, "can't open file")
	}
	defer fin.Close()

	mimeT := mime.TypeByExtension(filepath.Ext(path))
	w.Status(gemini.StatusSuccess, mimeT)
	io.Copy(w, fin)
}
