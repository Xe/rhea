package geminitest

import (
	"bytes"
)

type ResponseRecorder struct {
	StatusCode int
	Meta       string
	Body       *bytes.Buffer
}

func (rr *ResponseRecorder) Status(status int, meta string) {
	rr.StatusCode = status
	rr.Meta = meta
}

func (rr *ResponseRecorder) Write(data []byte) (int, error) {
	if rr.Body == nil {
		rr.Body = bytes.NewBuffer(nil)
	}

	return rr.Body.Write(data)
}
