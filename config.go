
package main

import (
	"net"
	"net/http"
)

type configEntry struct {
	ReadOnly	bool	`json:"readonly"`
	ReplaceImage	string	`json:"replaceimage"`
}

func getConfig(r *http.Request) configEntry {
	cfg := configEntry{
		ReadOnly: true,
		ReplaceImage: Config.Access.ReplaceImage,
	}
	ip := net.ParseIP(r.RemoteAddr)
	if ip != nil {
		for _, cidr := range Config.Access.ReadWrite {
			if (&cidr).Contains(ip) {
				cfg.ReadOnly = false
				break
			}
		}
	}
	return cfg
}

func ConfigHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		ConfigHandlerGet(w, r)
	case "OPTIONS":
		w.WriteHeader(200)
	default:
		http.Error(w, "Bad method", http.StatusMethodNotAllowed)
	}
	return
}

func ConfigHandlerGet(w http.ResponseWriter, r *http.Request) {
	cfg := getConfig(r)
	outputJSON(w, cfg, http.StatusOK)
}

