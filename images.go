
package main

import (
	"log"
	"time"
	"strconv"
	"encoding/json"
	"database/sql"
	"net/http"
	"github.com/gorilla/mux"
)

type imageEntry struct {
	Id		int64	`json:"id"`
	Created		int64	`json:"created"`
	IpAddress	string	`json:"ipaddress"`
	ClientUUID	string	`json:"clientuuid"`
	CharacterId	int64	`json:"characterId"`
	Rating		int64	`json:"rating"`
	Text1		string	`json:"text1"`
	Text2		string	`json:"text2"`
	Deleted		bool	`json:"deleted"`
}

type patchObject struct {
	Op	string	`json:"op"`
	Path	string	`json:"path"`
	Value	string	`json:"value"`
}

func scanImage(rows *sql.Rows, img *imageEntry) error {
	var deleted int
	err := rows.Scan(&img.Id, &img.Created, &img.IpAddress,
			&img.ClientUUID,&img.CharacterId, &img.Rating,
			&img.Text1, &img.Text2, &deleted)
	if deleted != 0 {
		img.Deleted = true
	}
	return err
}

func ImagesHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		ImagesHandlerGet(w, r)
	case "POST":
		ImagesHandlerPost(w, r)
	case "OPTIONS":
		w.WriteHeader(200)
	default:
		http.Error(w, "Bad method", http.StatusMethodNotAllowed)
	}
	return
}

func ImagesHandlerGet(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,created,ipaddress,clientuuid,character_id,rating,text1,text2,deleted FROM images WHERE deleted != 1")
	if err != nil {
		log.Printf("ImagesHandlerGet: SELECT: %s", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	res := make([]imageEntry, 0, 32)
	for rows.Next() {
		var img imageEntry
		if err := scanImage(rows, &img); err != nil {
			log.Printf("ImagesHandlerGet: SELECT: %s", err.Error())
			http.Error(w, err.Error(), 500)
			return
		}
		res = append(res, img)
	}
	outputJSON(w, res, http.StatusOK)
}

// add image.
func ImagesHandlerPost(w http.ResponseWriter, r *http.Request) {

	clientuuid, err := r.Cookie("clientuuid")
	if err != nil {
		http.Error(w, "missing clientuuid cookie", 406)
		return
	}

	// first decode incoming JSON
	var img imageEntry
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&img); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// see if it is complete.
	if img.CharacterId <= 0 ||
	   (img.Text1 == "" && img.Text2 == "") || img.Deleted {
		http.Error(w, "Unprocessable Entity", 422)
		return
	}

	sessCfg := getConfig(r)
	if sessCfg.ReadOnly {
		http.Error(w, "Forbidden", 403)
		return
	}

	// add to database
	tx, err := db.Begin()
	if err != nil {
		log.Printf("ImagesHandlerPost: db.Begin: %s", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}

	now := int64(time.Now().Unix())

	res, err := tx.Exec("INSERT INTO " +
		"images(created,ipaddress,clientuuid,character_id," +
		"	rating,text1,text2,deleted) " +
		"VALUES (?,?,?,?,?,?,?,?)",
		now, r.RemoteAddr, clientuuid, img.CharacterId, 0,
		img.Text1, img.Text2, 0)
	if err != nil {
		log.Printf("ImagesHandlerPost: INSERT: %s", err.Error())
		http.Error(w, err.Error(), 500)
		tx.Rollback()
		return
	}
	id, _ := res.LastInsertId()

	err = tx.Commit()
	if err != nil {
		log.Printf("ImagesHandlerPost: COMMIT: %s", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}

	// return new object
	ImageHandlerGet(w, r, id, http.StatusCreated)
}

func ImageHandlerGet(w http.ResponseWriter, r *http.Request, idnum int64, code int) {

	rows, err := db.Query("SELECT id,created,ipaddress,clientuuid,character_id,rating,text1,text2,deleted FROM images WHERE id=?", idnum)
	if err != nil {
		log.Printf("ImagesHandlerGet: SELECT: %s", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var img imageEntry
		if err := scanImage(rows, &img); err != nil {
			log.Printf("ImagesHandlerGet: SELECT: %s", err.Error())
			http.Error(w, err.Error(), 500)
			return
		}
		if code == 0 {
			code = http.StatusOK
		}
		outputJSON(w, img, code)
		return
	}
	http.NotFound(w, r)
}

func ImageHandlerPatch(w http.ResponseWriter, r *http.Request, idnum int64) {

	// first decode incoming JSON patch operation
	var patch patchObject
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&patch); err != nil {
		// fmt.Printf("decode error: %s\n", err)
		http.Error(w, err.Error(), 500)
		return
	}

	if patch.Op != "delta" ||
	   patch.Path != "/rating" || patch.Value == "" {
		http.Error(w, "Unprocessable Entity", 422)
		return
	}
	delta, _ := strconv.ParseInt(patch.Value, 10, 64)
	if delta > 1 { delta = 1 }
	if delta < -1 { delta = -1 }

	// get clientuuid from cookie.
	cookie, err := r.Cookie("clientuuid"); if err != nil {
		http.Error(w, "missing clientuuid cookie", 406)
		return
	}
	clientuuid := cookie.Value

	tx, err := db.Begin()
	if err != nil {
		log.Printf("ImagesHandlerPatch: db.Begin: %s", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}

	var rating int64
	// Get current rating
	row := db.QueryRow("SELECT rating FROM images WHERE id=?", idnum)
	err = row.Scan(&rating)
	switch {
		case err == sql.ErrNoRows:
			tx.Rollback()
			http.NotFound(w, r)
			return
		case err != nil:
			tx.Rollback()
			log.Printf("ImagesHandlerPatch: SELECT: %s", err.Error())
			http.Error(w, err.Error(), 500)
			return
	}

	// See if this user has already rated the image.
	var rated int64
	row = db.QueryRow("SELECT rated FROM ratedimages " +
			  "WHERE image_id=? AND clientuuid=?",
				idnum, clientuuid)
	err = row.Scan(&rated)
	switch {
		case err == sql.ErrNoRows:
			break
		case err != nil:
			tx.Rollback()
			log.Printf("ImagesHandlerPatch: SELECT: %s", err.Error())
			http.Error(w, err.Error(), 500)
			return
	}

	if rated != 0 {
		// already rated. can we "unrate" (up <=> down) ?
		if rated == delta {
			// nope. return unmodified object.
			ImageHandlerGet(w, r, idnum, http.StatusOK)
			return
		}
		_, err = tx.Exec("UPDATE ratedimages SET rated = ? " +
				 "WHERE clientuuid=? AND image_id=?",
				delta, clientuuid, idnum)
		if err != nil {
			log.Printf("ImagesHandlerPatch: UPDATE: %s",
					err.Error())
		}
		delta *= 2
	} else {
		// new rating.
		_, err = tx.Exec("INSERT INTO ratedimages" +
				 "(clientuuid, image_id, rated) " +
				"VALUES(?,?,?)",
				clientuuid, idnum, delta)
		if err != nil {
			log.Printf("ImagesHandlerPatch: INSERT: %s",
					err.Error())
		}
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		tx.Rollback()
		return
	}

	rating += delta
	_, err = tx.Exec("UPDATE images SET rating = ? WHERE id=?",
				rating, idnum)
	if err != nil {
		log.Printf("ImagesHandlerPatch: UPDATE: %s", err.Error())
		http.Error(w, err.Error(), 500)
		tx.Rollback()
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("ImagesHandlerPatch: COMMIT: %s", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}

	// return updated object
	ImageHandlerGet(w, r, idnum, http.StatusOK)
}

func ImageHandlerPut(w http.ResponseWriter, r *http.Request, idnum int64) {
	http.Error(w, "Bad method", http.StatusMethodNotAllowed)
}

func ImageHandlerDelete(w http.ResponseWriter, r *http.Request, idnum int64) {
	// update database
	tx, err := db.Begin()
	if err != nil {
		log.Printf("ImagesHandlerDelete: db.Begin: %s", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}

	res, err := tx.Exec("UPDATE images SET deleted = 1 WHERE id=?", idnum)
	if err != nil {
		log.Printf("ImagesHandlerDelete: UPDATE: %s", err.Error())
		http.Error(w, err.Error(), 500)
		tx.Rollback()
		return
	}
	count, _ := res.RowsAffected()

	err = tx.Commit()
	if err != nil {
		log.Printf("ImagesHandlerDelete: COMMIT: %s", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}

	if count == 0 {
		http.NotFound(w, r)
		return
	}

	// return new object
	ImageHandlerGet(w, r, idnum, http.StatusCreated)
}

func ImageHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	var idnum int64 = -1
	if idstr, ok := vars["id"]; ok {
		idnum, _ = strconv.ParseInt(idstr, 10, 64)
	}

	switch r.Method {
	case "GET":
		ImageHandlerGet(w, r, idnum, http.StatusOK)
	case "PUT":
		ImageHandlerPut(w, r, idnum)
	case "DELETE":
		ImageHandlerDelete(w, r, idnum)
	case "PATCH":
		ImageHandlerPatch(w, r, idnum)
	case "OPTIONS":
		w.WriteHeader(200)
	default:
		http.Error(w, "Bad method", http.StatusMethodNotAllowed)
	}
}

