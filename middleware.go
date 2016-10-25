package main

import (
	"fmt"
//	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

// statusWriter proxies http.ResponseWriter
// and stores the requests status and length.
type statusWriter struct {
	http.ResponseWriter
	status int
	length int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	w.length = len(b)
	return w.ResponseWriter.Write(b)
}

// HttpLog calls ServeHTTP with a custom responsewriter that
// stores the requests status and length so we can log it.
func HttpLog(handle http.Handler) http.HandlerFunc {
	if handle == nil {
		handle = http.DefaultServeMux;
	}
	return func(w http.ResponseWriter, request *http.Request) {
		start := time.Now()
		writer := statusWriter{w, 0, 0}
		url := request.URL.String()

		// change RemoteAddr if proxied
		xff := request.Header.Get("X-Forwarded-For")
		if xff != "" {
			request.RemoteAddr = xff
		}

		// copy x-cookie to cookie
		xk := request.Header.Get("X-Cookie")
		if xk != "" {
			request.Header.Add("Cookie", xk)
		}

		// remove port from request
		host, _, err := net.SplitHostPort(request.RemoteAddr)
		if err == nil {
			request.RemoteAddr = host
		}

		// Always set Access-Control-Allow-Origin header
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods",
			"DELETE, GET, OPTIONS, POST, PUT, PATCH")
		w.Header().Set("Access-Control-Allow-Headers",
				"Content-Type, X-Cookie")

		handle.ServeHTTP(&writer, request)
		end := time.Now()
		latency := end.Sub(start)

		fmt.Printf("%v %s %s \"%s %s %s\" %d %d %s %v\n",
			end.Format("2006/01/02 15:04:05"),
			request.Host,
			request.RemoteAddr,
			request.Method,
			url,
			request.Proto,
			writer.status,
			writer.length,
			strconv.Quote(request.Header.Get("User-Agent")),
			latency)
	}
}
