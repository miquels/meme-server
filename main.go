package main

import (
	"fmt"
	"log"
	"net"

	"database/sql"
	"encoding/json"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	"github.com/gorilla/mux"
	"github.com/XS4ALL/curlyconf-go"
)

var configFile = "/home/mikevs/go/server/server.cfg"

type cfgMain struct {
	HtmlDir	string
	FilesDir string
	DbDir	string
	Access	cfgAccess
}

type cfgAccess struct {
	Proxy		[]net.IPNet
	ReadWrite	[]net.IPNet
	ReadOnly	[]net.IPNet
	ReplaceImage	string
}
var Config cfgMain

var db *sql.DB

func main() {

	p, err := curlyconf.NewParser(configFile, curlyconf.ParserNL)
	if err == nil {
		err = p.Parse(&Config)
	}
	if err != nil {
		fmt.Println(err.(*curlyconf.ParseError).LongError())
		return
	}

	db, err = sql.Open("sqlite3", Config.DbDir + "/ripememes.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	router := mux.NewRouter()

	router.HandleFunc("/api/config", ConfigHandler)
	router.HandleFunc("/api/images", ImagesHandler)
	router.HandleFunc("/api/images/{id:[0-9]+}", ImageHandler)
	router.HandleFunc("/api/characters", CharactersHandler)
	router.HandleFunc("/api/characters/{id:[0-9]+}", CharacterHandler)
	router.HandleFunc("/api/files/{name}", FilesHandler)

	var indexHandler = http.FileServer(http.Dir(Config.HtmlDir))
	router.Handle("/{name}", indexHandler)
	router.Handle("/", indexHandler)

	log.Fatal(http.ListenAndServe(":8040", HttpLog(router)))
}

func outputJSON(w http.ResponseWriter, res interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if code == 0 {
		code = http.StatusOK
	}
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		panic(err)
	}
}

