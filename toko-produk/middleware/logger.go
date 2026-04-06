package middleware

import (
	"log"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
		size:           0,
	}
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		warpped := newResponseWriter(w)
		next.ServeHTTP(warpped, r)

		duration := time.Since(start)

		var label string
		switch {
		case warpped.status >= 500:
			label = "error"
		case warpped.status >= 400:
			label = "client error"
		case warpped.status >= 300:
			label = "redirect"
		case warpped.status >= 200:
			label = "success"
		}
		log.Printf("%s %s %s %d %d %s", label, r.Method, r.URL.RequestURI(), warpped.status, warpped.size, duration)

	})
}
