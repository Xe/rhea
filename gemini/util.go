package gemini

// NotFound is a generic handler for when you can't find a resource.
func NotFound() Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		w.Status(StatusNotFound, r.URL.Path+" not found")
	})
}
