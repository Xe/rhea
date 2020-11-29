package gemini

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var requestCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "gemini_requests",
	Help: "The number of gemini requests handled",
}, []string{"domain", "status"})

// Server is a gemini server struct in the vein of net/http#Server.
type Server struct {
	lis net.Listener
	hdl Handler
}

func NewServer(hdl Handler) *Server {
	return &Server{
		hdl: hdl,
	}
}

func (s *Server) ListenAndServe(addr string, certPath string, keyPath string) error {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return err
	}

	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	lis, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		return err
	}

	return s.Serve(lis)
}

func (s *Server) Serve(lis net.Listener) error {
	if s.lis != nil {
		return fmt.Errorf("listener already set")
	}

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Printf("listening error: %v", err)
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	cw := connWrapper{Conn: conn}
	r := bufio.NewReader(io.LimitReader(conn, 1024))
	tpr := textproto.NewReader(r)
	uText, err := tpr.ReadLine()
	if err != nil {
		log.Printf("can't read request from %s: %v", conn.RemoteAddr().String(), err)
		cw.Status(StatusBadRequest, "invalid request")
		return
	}

	u, err := url.Parse(uText)
	if err != nil {
		log.Printf("can't read url from %s: %v", conn.RemoteAddr().String(), err)
		cw.Status(StatusBadRequest, "invalid url")
		return
	}

	cw.domain = u.Host

	req := &Request{
		URL: u,
	}

	if tc, ok := conn.(*tls.Conn); ok {
		if certs := tc.ConnectionState().PeerCertificates; len(certs) != 0 {
			req.Cert = certs[0]
		}
	}

	s.hdl.HandleGemini(cw, req)
}

type Request struct {
	URL  *url.URL
	Cert *x509.Certificate
}

type ResponseWriter interface {
	Status(status int, meta string)
	io.Writer
}

type connWrapper struct {
	net.Conn
	domain string
}

func (cw connWrapper) Status(status int, meta string) {
	requestCount.With(
		prometheus.Labels{
			"domain": cw.domain,
			"status": fmt.Sprint(status),
		},
	).Inc()

	fmt.Fprintf(cw, "%d %s\r\n", status, meta)
}

type Handler interface {
	HandleGemini(rw ResponseWriter, req *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (hf HandlerFunc) HandleGemini(w ResponseWriter, r *Request) { hf(w, r) }
