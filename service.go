// Copyright (c) 2012-2014 José Carlos Nieto, https://menteslibres.net/xiam
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
	"log"
	"net"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/xiam/hyperfox/tools/capture"
	"menteslibres.net/gosexy/to"
	"upper.io/db"
)

var (
	cleanPattern  = regexp.MustCompile(`[^0-9a-zA-Z\s\.]`)
	spacesPattern = regexp.MustCompile(`\s+`)
)

const (
	serviceBindHost      = `127.0.0.1`
	serviceBindStartPort = 3030
)

type getResponse struct {
	capture.Response `json:",inline"`
}

func (g getResponse) Constraint() db.Cond {
	return db.Cond{"id": g.ID}
}

type pullResponse struct {
	Data  []capture.Response `json:"data"`
	Pages uint               `json:"pages"`
	Page  uint               `json:"page"`
}

func replyJSON(wri http.ResponseWriter, data interface{}) {
	var buf []byte
	var err error

	if buf, err = json.Marshal(data); err != nil {
		log.Printf("Marshal: %q", err)
		return
	}

	wri.Header().Set("Access-Control-Allow-Origin", "*")

	wri.WriteHeader(http.StatusOK)
	wri.Write(buf)
}

const (
	pageSize = uint(10)
)

func rootHandler(wri http.ResponseWriter, req *http.Request) {

}

// downloadHandler provides a downloadable document given its ID.
func downloadHandler(wri http.ResponseWriter, req *http.Request) {
	var err error
	var response getResponse

	if err = req.ParseForm(); err != nil {
		log.Printf("ParseForm: %q", err)
		return
	}

	response.ID = uint(to.Int64(req.Form.Get("id")))

	res := col.Find(response)

	res.Select(
		"id",
		"url",
		db.Raw{"hex(body) AS body"},
		"date",
	)

	if err = res.One(&response.Response); err != nil {
		log.Printf("res.One: %q", err)
		return
	}

	var buf []byte
	buf, _ = hex.DecodeString(string(response.Body))

	http.ServeContent(wri, req, path.Base(response.URL), response.DateEnd, bytes.NewReader(buf))

	return
}

// getHandler service returns a request body.
func getHandler(wri http.ResponseWriter, req *http.Request) {
	var err error
	var response getResponse

	if err = req.ParseForm(); err != nil {
		log.Printf("ParseForm: %q", err)
		return
	}

	response.ID = uint(to.Int64(req.Form.Get("id")))

	res := col.Find(response)

	res.Select(
		"id",
		"method",
		"remote_addr",
		"status",
		"host",
		"url",
		"header",
		"date",
	)

	if err = res.One(&response.Response); err != nil {
		log.Printf("res.One: %q", err)
		return
	}

	replyJSON(wri, response)

	return
}

// pullHandler service serves paginated requests.
func pullHandler(wri http.ResponseWriter, req *http.Request) {
	var err error
	var response pullResponse

	if err = req.ParseForm(); err != nil {
		log.Printf("ParseForm: %q", err)
		return
	}

	q := req.Form.Get("q")

	q = cleanPattern.ReplaceAllString(q, " ")
	q = spacesPattern.ReplaceAllString(q, " ")

	response.Page = uint(to.Int64(req.Form.Get("page")))

	if response.Page < 1 {
		response.Page = 1
	}

	// Result set
	res := col.Find().Select(
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
		"date_end",
		"time_taken",
	).Sort("id").Limit(pageSize).Skip(pageSize * (response.Page - 1))

	if q != "" {

		terms := strings.Split(q, " ")
		conds := make(db.Or, 0, len(terms))

		for _, term := range terms {
			conds = append(conds, db.Or{
				db.Raw{`host LIKE '%` + term + `%'`},
				db.Raw{`origin LIKE '%` + term + `%'`},
				db.Raw{`path LIKE '%` + term + `%'`},
				db.Raw{`content_type LIKE '%` + term + `%'`},
				db.Cond{"method": term},
				db.Cond{"scheme": term},
				db.Cond{"status": term},
			},
			)
		}

		res.Where(conds...)
	}

	// Pulling information page.
	if err = res.All(&response.Data); err != nil {
		log.Printf("res.All: %q", err)
		return
	}

	// Getting total number of pages.
	if c, err := res.Count(); err == nil {
		response.Pages = uint(c) / pageSize
	}

	replyJSON(wri, response)

	return
}

// startServices starts an http server that provides websocket and rest
// services.
func startServices() error {

	r := mux.NewRouter()

	r.HandleFunc("/", rootHandler)
	r.HandleFunc("/pull", pullHandler)
	r.HandleFunc("/get", getHandler)
	r.HandleFunc("/download", downloadHandler)

	errc := make(chan error)

	go func(errc chan error) {

		var err error
		var addr string
		var ln net.Listener

		// Looking for a port to listen to.
		for i := 0; i < 65535; i++ {
			addr = serviceBindHost + ":" + strconv.Itoa(serviceBindStartPort+i)
			if ln, err = net.Listen("tcp", addr); err == nil {
				// We have a listener!
				break
			}
			if strings.Contains(err.Error(), "address already in use") == false {
				// We don't know how to handle this error.
				errc <- err
				return
			}
		}

		log.Printf("See status at http://%s/", addr)

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

		errc <- nil

	}(errc)

	err := <-errc

	return err
}
