package gemini

import (
	"net/url"
	"testing"
)

func TestServeMuxHandler(t *testing.T) {
	mux := NewServeMux()
	mux.Handle("/", NotFound())

	u, err := url.Parse("gemini://foo.localhost/")
	if err != nil {
		t.Fatal(err)
	}

	h, pat := mux.Handler(&Request{
		URL: u,
	})
	_ = h

	if pat == "" {
		t.Fatal("wanted a returned pattern, got nothing")
	}
}
