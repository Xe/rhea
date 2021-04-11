package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Xe/rhea/gemini"
)

type ReverseProxy struct {
	To     []string `json:"to"`
	Domain string   `json:"domain"`
}

func (rp ReverseProxy) HandleGemini(w gemini.ResponseWriter, r *gemini.Request) {
	target := rp.To[rand.Intn(len(rp.To))]
	u, _ := url.Parse(target)
	var conn net.Conn
	var err error

	switch u.Scheme {
	case "unix":
		conn, err = net.Dial("unix", filepath.Join("/", u.Host, u.Path))
	case "tcp":
		conn, err = net.Dial("tcp", u.Host)
	case "tls":
		conn, err = tls.Dial(
			"tcp",
			u.Host,
			&tls.Config{InsecureSkipVerify: true},
		)
	}

	if err != nil {
		w.Status(gemini.StatusProxyError, err.Error())
		return
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	r.URL.Host = rp.Domain
	fmt.Fprintf(conn, "%s\r\n", r.URL.String())
	buf := bufio.NewReader(conn)
	statusLineBytes, _, err := buf.ReadLine()
	sp := strings.SplitN(string(statusLineBytes), " ", 2)

	status, err := strconv.Atoi(sp[0])
	if err != nil {
		w.Status(gemini.StatusTemporaryFailure, err.Error())
		return
	}
	meta := sp[1]
	w.Status(status, meta)

	io.Copy(w, conn)
}
