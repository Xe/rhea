package main

import (
	"crypto/tls"
	_ "embed"
	"fmt"
	"net"
	"net/url"
	"os"
	"testing"

	"github.com/Xe/rhea/gemini"
	"github.com/Xe/rhea/gemini/geminitest"
)

type testHandler struct{}

func (testHandler) HandleGemini(w gemini.ResponseWriter, r *gemini.Request) {
	if r.URL.Host != "test.server" {
		w.Status(gemini.StatusBadRequest, "cannot proxy request")
	}

	w.Status(gemini.StatusSuccess, "text/gemini")
	fmt.Fprintln(w, `# test

This is a test.`)
}

//go:embed testdata/cert.pem
var certPem []byte

//go:embed testdata/key.pem
var keyPem []byte

func TestReverseProxy(t *testing.T) {
	t.Run("unix socket", func(t *testing.T) {
		f, err := os.CreateTemp("", "rhea")
		if err != nil {
			t.Fatal(err)
		}
		fname := f.Name()
		f.Close()

		os.Remove(fname)
		l, err := net.Listen("unix", fname)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		s := gemini.NewServer(testHandler{})
		go s.Serve(l)

		rp := ReverseProxy{
			To:     []string{"unix://" + fname},
			Domain: "test.server",
		}
		u, _ := url.Parse("gemini://foo.local")

		rw := new(geminitest.ResponseRecorder)
		rp.HandleGemini(rw, &gemini.Request{URL: u})

		if rw.StatusCode != gemini.StatusSuccess {
			t.Fatalf("wanted status code %d, got: %d", gemini.StatusSuccess, rw.StatusCode)
		}
	})

	t.Run("tcp socket", func(t *testing.T) {
		l, err := net.Listen("tcp", "[::1]:0")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		s := gemini.NewServer(testHandler{})
		go s.Serve(l)

		rp := ReverseProxy{
			To:     []string{"tcp://" + l.Addr().String()},
			Domain: "test.server",
		}
		u, _ := url.Parse("gemini://foo.local")

		rw := new(geminitest.ResponseRecorder)
		rp.HandleGemini(rw, &gemini.Request{URL: u})

		if rw.StatusCode != gemini.StatusSuccess {
			t.Fatalf("wanted status code %d, got: %d", gemini.StatusSuccess, rw.StatusCode)
		}
	})

	t.Run("tls socket", func(t *testing.T) {
		cert, err := tls.X509KeyPair(certPem, keyPem)
		if err != nil {
			t.Fatal(err)
		}
		cfg := &tls.Config{Certificates: []tls.Certificate{cert}}

		l, err := tls.Listen("tcp", "[::1]:0", cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		s := gemini.NewServer(testHandler{})
		go s.Serve(l)

		rp := ReverseProxy{
			To:     []string{"tls://" + l.Addr().String()},
			Domain: "test.server",
		}
		u, _ := url.Parse("gemini://foo.local")

		rw := new(geminitest.ResponseRecorder)
		rp.HandleGemini(rw, &gemini.Request{URL: u})

		if rw.StatusCode != gemini.StatusSuccess {
			t.Fatalf("wanted status code %d, got: %d", gemini.StatusSuccess, rw.StatusCode)
		}
	})
}
