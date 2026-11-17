package api

import (
	"net"
	"net/http"
	"net/url"
)

// anyone lets every request in. It only exists behind LocalOnly, which is
// why it is not exported.
type anyone struct{}

func (anyone) Authenticate(*http.Request) bool { return true }

// LocalOnly protects a server that listens on the loopback and has no
// login. Listening on 127.0.0.1 keeps other machines out, but not the
// browser of this one: any site the person visits can make the browser
// call http://127.0.0.1. So a request only passes when it was addressed to
// this machine by a loopback name and, if a page made it, the page is this
// server's own.
//
//   - Host must be localhost or a loopback address. A site that points its
//     own domain at 127.0.0.1 (DNS rebinding) arrives with its domain here.
//   - Origin, when the browser sends it, must be this same server. Browsers
//     send it on everything that is not a plain GET.
//   - Sec-Fetch-Site, when the browser sends it, must say the request came
//     from the same origin, or from the person (typing the address).
//
// Programs on the machine (curl) send none of the last two and pass: they
// already run as the owner.
func LocalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !localRequest(r) {
			writeError(w, http.StatusForbidden, "pedido recusado: o painel só atende a própria página, nesta máquina")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func localRequest(r *http.Request) bool {
	if !loopbackHost(r.Host) {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "http" || u.Host != r.Host {
			return false
		}
	}
	switch r.Header.Get("Sec-Fetch-Site") {
	case "", "same-origin":
		return true
	case "none":
		// The person typed the address or used a bookmark. That only ever
		// reads.
		return r.Method == http.MethodGet || r.Method == http.MethodHead
	}
	return false
}

// loopbackHost reports whether a Host header names this machine.
func loopbackHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport // no port
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
