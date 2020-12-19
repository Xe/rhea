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

	"github.com/Xe/rhea/limitwriter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var requestCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "gemini_requests_total",
	Help: "The number of gemini requests handled",
}, []string{"domain", "status"})

// Server is a gemini server struct in the vein of net/http#Server.
type Server struct {
	lis net.Listener
	hdl Handler
}

// NewServer creates a new Gemini server based on a particular handler.
func NewServer(hdl Handler) *Server {
	return &Server{
		hdl: hdl,
	}
}

// ListenAndServe creates a new TLS listener on a given port with a given TLS certificate
// and key. This is most useful for serving a single site. If you need more control or
// want to host multiple sites, create your own tls.Listener and use the Serve method.
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

// Serve serves gemini responses to clients connecting to this Listener.
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

	cw := &connWrapper{Writer: limitwriter.New(conn, 4*1024*1024)}
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

	if u.Path == "" {
		u.Path = "/"
	}

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

// Request contains all relevant metadata for a gemini request.
type Request struct {
	URL  *url.URL
	Cert *x509.Certificate
}

// ResponseWriter is used by a gemini handler to construct a gemini response.
//
// This may not be used after the HandleGemini method has returned.
type ResponseWriter interface {
	// Status sends the status line to the client with the provided status
	// code and metadata.
	//
	// Only one status line may be sent to the client. Making more than one
	// call to this method should result in a panic.
	//
	// The provided code SHOULD be one of the status codes defined in this
	// package.
	Status(status int, meta string)
	io.Writer
}

type connWrapper struct {
	io.Writer
	status int
	domain string
}

func (cw *connWrapper) Status(status int, meta string) {
	if cw.status != 0 {
		panic("Status called twice")
	}

	requestCount.With(
		prometheus.Labels{
			"domain": cw.domain,
			"status": fmt.Sprint(status),
		},
	).Inc()

	fmt.Fprintf(cw, "%d %s\r\n", status, meta)
	cw.status = status
}

// A Handler responds to a gemini request.
//
// HandleGemini should write the status line and any relevant body
// data then return.
//
// The provided request is safe to modify provided you really
// understand what you are doing.
type Handler interface {
	HandleGemini(rw ResponseWriter, req *Request)
}

// HandlerFunc is a convenience wrapper that lets you easily construct
// Handlers from ad-hoc functions.
type HandlerFunc func(ResponseWriter, *Request)

func (hf HandlerFunc) HandleGemini(w ResponseWriter, r *Request) { hf(w, r) }
