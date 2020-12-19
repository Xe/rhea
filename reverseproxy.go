package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"

	"github.com/Xe/rhea/gemini"
)

type ReverseProxy struct {
	To     []string `json:"to"`
	Domain string   `json:"domain"`
}

func (rp ReverseProxy) HandleGemini(w gemini.ResponseWriter, r *gemini.Request) {
	conn, err := tls.Dial(
		"tcp",
		rp.To[rand.Intn(len(rp.To))],
		&tls.Config{InsecureSkipVerify: true},
	)
	if err != nil {
		w.Status(gemini.StatusProxyError, err.Error())
		return
	}
	defer conn.Close()

	r.URL.Host = rp.Domain
	fmt.Fprintf(conn, "%s\r\n", r.URL.String())
	io.Copy(w, conn)
}
