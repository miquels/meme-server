
package main

import (
	"regexp"
	"io/ioutil"
	"strings"
	"strconv"
//	"database/sql"
	"net/http"
	"github.com/gorilla/mux"
)

type characterEntry struct {
	Id	int64	`json:"id"`
	Name	string	`json:"name"`
	Url	string	`json:"url"`
	Rating	uint	`json:"rating"`
}

var characterRe *regexp.Regexp =
	regexp.MustCompile(`^0*(\d+)\.(.*)\.(jpg|jpeg|png|gif)$`)

func CharactersHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		break
	case "OPTIONS":
		w.WriteHeader(200)
		return
	default:
		http.Error(w, "Bad method", http.StatusMethodNotAllowed)
		return
	}

	d, err := ioutil.ReadDir(Config.FilesDir)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	vars := mux.Vars(r)
	var idnum int64 = -1
	if idstr, ok := vars["id"]; ok {
		idnum, _ = strconv.ParseInt(idstr, 10, 64)
	}

	e := make([]characterEntry, 0, 32)
	var res interface{}
	for _, fi := range d {
		name := fi.Name()
		s := characterRe.FindStringSubmatch(name)
		if s == nil {
			continue
		}
		id, _ := strconv.ParseInt(s[1], 10, 64)
		e = append(e, characterEntry{
			Id:	id,
			Name:	strings.Replace(s[2], ".", " ", -1),
			Url:	"/api/files/" + name,
		})
		if idnum >= 0 && idnum == id {
			res = e[len(e) - 1]
			break
		}
	}

	if idnum >= 0 {
		if res == nil {
			http.NotFound(w, r)
			return
		}
	} else {
		res = e
	}

	outputJSON(w, res, http.StatusOK)
}

func CharacterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		CharactersHandler(w, r)
	case "OPTIONS":
		w.WriteHeader(200)
	default:
		http.Error(w, "Bad method", http.StatusMethodNotAllowed)
	}
}

