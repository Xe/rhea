package main

import (
	"crypto/tls"
	"fmt"
	"log"

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
	if r.URL.Scheme != "gemini" {
		w.Status(gemini.StatusProxyRequestRefused, fmt.Sprintf("can't proxy to %s", r.URL.Host))
	}

	host := r.URL.Hostname()

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

	if s.ReverseProxy != nil {
		s.ReverseProxy.HandleGemini(w, r)
		return
	}

	w.Status(gemini.StatusUnavailable, "no active configuration detected")
	log.Printf("no active configuration domain=%s", r.URL.Hostname())
}
