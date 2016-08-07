// Copyright (c) 2012-today José Carlos Nieto, https://menteslibres.net/xiam
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/xiam/hyperfox/lib/plugins/capture"
	"menteslibres.net/gosexy/to"
	"upper.io/db.v2"
)

var (
	cleanPattern  = regexp.MustCompile(`[^0-9a-zA-Z\s\.]`)
	spacesPattern = regexp.MustCompile(`\s+`)
)

const (
	serviceBindHost      = `127.0.0.1`
	serviceBindStartPort = 3030
)

const (
	pageSize         = 50
	directionRequest = `req`
)

type getResponse struct {
	capture.Response `json:",inline"`
}

type pullResponse struct {
	Data  []capture.Response `json:"data"`
	Pages uint               `json:"pages"`
	Page  uint               `json:"page"`
}

func replyCode(w http.ResponseWriter, httpCode int) {
	w.WriteHeader(httpCode)
	w.Write([]byte(http.StatusText(httpCode)))
}

func replyJSON(w http.ResponseWriter, data interface{}) {
	var buf []byte
	var err error

	if buf, err = json.Marshal(data); err != nil {
		log.Printf("Marshal: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.WriteHeader(http.StatusOK)
	w.Write(buf)
}

func rootHandler(http.ResponseWriter, *http.Request) {

}

// downloadHandler provides a downloadable document given its ID.
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	if err = r.ParseForm(); err != nil {
		log.Printf("ParseForm: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	wireFormat := to.Bool(r.Form.Get("wire"))
	direction := r.Form.Get("type")

	var response getResponse

	res := storage.Find(response.ID)

	res.Select(
		"id",
		"url",
		"method",
		"header",
		"request_header",
		"date_end",
		db.Raw("hex(body) AS body"),
		db.Raw("hex(request_body) AS request_body"),
	)

	if err = res.One(&response.Response); err != nil {
		log.Printf("res.One: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	var u *url.URL
	if u, err = url.Parse(response.URL); err != nil {
		log.Printf("url.Parse: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	var body []byte
	basename := path.Base(u.Path)
	var headers http.Header

	if direction == directionRequest {
		if body, err = hex.DecodeString(string(response.RequestBody)); err != nil {
			log.Printf("url.Parse: %q", err)
			replyCode(w, http.StatusInternalServerError)
			return
		}
		headers = response.RequestHeader.Header
	} else {
		if body, err = hex.DecodeString(string(response.RequestBody)); err != nil {
			log.Printf("url.Parse: %q", err)
			replyCode(w, http.StatusInternalServerError)
			return
		}
		headers = response.Header.Header
	}

	if wireFormat {
		buf := bytes.NewBuffer(nil)
		for k, vv := range headers {
			for _, v := range vv {
				buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
			}
		}
		buf.WriteString("\r\n")
		w.Header().Set("Content-Disposition", `attachment; filename="`+u.Host+"-"+basename+`.bin"`)
		buf.Write(body)

		http.ServeContent(w, r, "", response.DateEnd, bytes.NewReader(buf.Bytes()))
		return
	}

	w.Header().Set("Content-Disposition", `attachment; filename="`+basename+`"`)
	http.ServeContent(w, r, basename, response.DateEnd, bytes.NewReader(body))
}

// getHandler service returns a request body.
func getHandler(w http.ResponseWriter, r *http.Request) {

	var err error
	if err = r.ParseForm(); err != nil {
		log.Printf("ParseForm: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	var response getResponse
	response.ID = uint(to.Int64(r.Form.Get("id")))

	res := storage.Find(response.ID)

	res.Select(
		"id",
		"method",
		"origin",
		"content_type",
		"content_length",
		"status",
		"host",
		"url",
		"scheme",
		"header",
		"request_header",
		"date_start",
		"date_end",
		"time_taken",
	)

	if err = res.One(&response.Response); err != nil {
		log.Printf("res.One: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	replyJSON(w, response)
}

// pullHandler service serves paginated requests.
func pullHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var response pullResponse

	if err = r.ParseForm(); err != nil {
		log.Printf("ParseForm: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	q := r.Form.Get("q")

	q = cleanPattern.ReplaceAllString(q, " ")
	q = spacesPattern.ReplaceAllString(q, " ")

	response.Page = uint(to.Int64(r.Form.Get("page")))

	if response.Page < 1 {
		response.Page = 1
	}

	// Result set
	res := storage.Find().Select(
		"id",
		"method",
		"origin",
		"status",
		"host",
		"path",
		"scheme",
		"url",
		"content_length",
		"content_type",
		"date_start",
		"time_taken",
	).OrderBy("id").Limit(pageSize).Offset(pageSize * int(response.Page-1))

	if q != "" {
		terms := strings.Split(q, " ")
		conds := db.Or()

		for _, term := range terms {
			conds.Or(
				db.Or(
					db.Raw(`host LIKE '%`+term+`%'`),
					db.Raw(`origin LIKE '%`+term+`%'`),
					db.Raw(`path LIKE '%`+term+`%'`),
					db.Raw(`content_type LIKE '%`+term+`%'`),
					db.Cond{"method": term},
					db.Cond{"scheme": term},
					db.Cond{"status": term},
				),
			)
		}

		res.Where(conds)
	}

	// Pulling information page.
	if err = res.All(&response.Data); err != nil {
		log.Printf("res.All: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	// Getting total number of pages.
	if c, err := res.Count(); err == nil {
		response.Pages = uint(math.Ceil(float64(c) / float64(pageSize)))
	}

	replyJSON(w, response)
}

// startServices starts an http server that provides websocket and rest
// services.
func startServices() error {

	r := mux.NewRouter()

	r.HandleFunc("/", rootHandler)
	r.HandleFunc("/pull", pullHandler)
	r.HandleFunc("/get", getHandler)
	r.HandleFunc("/download", downloadHandler)

	log.Printf("Starting (local) API server...")

	// Looking for a port to listen to.
	ln, err := net.Listen("tcp", serviceBindHost+":0")
	if err != nil {
		log.Fatal("net.Listen: ", err)
	}

	addr := fmt.Sprintf("%s:%d", serviceBindHost, ln.Addr().(*net.TCPAddr).Port)
	log.Printf("Watch live capture at http://live.hyperfox.org/#/?source=%s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Serving API.
	go func() {
		if err := srv.Serve(ln); err != nil {
			panic(err.Error())
		}
	}()

	return err
}
