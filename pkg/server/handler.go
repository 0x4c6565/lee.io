package server

import (
	"fmt"
	"net/http"
	"strings"
)

var xForwardedFor string = http.CanonicalHeaderKey("X-Forwarded-For")
var xForwardedPort string = http.CanonicalHeaderKey("X-Forwarded-Port")

func ProxyHeaders(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(xForwardedFor) != "" && r.Header.Get(xForwardedPort) != "" {
			fwd := r.Header.Get(xForwardedFor)
			s := strings.Index(fwd, ", ")
			if s == -1 {
				s = len(fwd)
			}
			host := fwd[:s]
			if strings.Contains(host, ":") {
				host = fmt.Sprintf("[%s]", host)
			}
			r.RemoteAddr = fmt.Sprintf("%s:%s", host, r.Header.Get(xForwardedPort))
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
