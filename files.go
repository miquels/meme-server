
package main

import (
	"net/http"
	"github.com/gorilla/mux"
)

func FilesHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		FilesHandlerGet(w, r)
	case "OPTIONS":
		w.WriteHeader(200)
	default:
		http.Error(w, "Bad method", http.StatusMethodNotAllowed)
	}
	return
}

func FilesHandlerGet(w http.ResponseWriter, r *http.Request) {

	sessCfg := getConfig(r)
	vars := mux.Vars(r)
	name, _ := vars["name"]

	if sessCfg.ReadOnly && sessCfg.ReplaceImage != "" &&
	   sessCfg.ReplaceImage != r.URL.Path {
		http.Redirect(w, r, sessCfg.ReplaceImage, 302)
	} else {
		http.ServeFile(w, r, Config.FilesDir + "/" + name)
	}
}

