// Copyright (c) 2012-today José Nieto, https://xiam.io
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
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/malfunkt/hyperfox/pkg/plugins/capture"
	_ "github.com/malfunkt/hyperfox/ui/statik"
	"github.com/pkg/browser"
	"github.com/rakyll/statik/fs"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"upper.io/db.v3"
)

var (
	reUnsafeChars   = regexp.MustCompile(`[^0-9a-zA-Z\s\.]`)
	reUnsafeFile    = regexp.MustCompile(`[^0-9a-zA-Z-_]`)
	reRepeatedDash  = regexp.MustCompile(`-+`)
	reRepeatedBlank = regexp.MustCompile(`\s+`)
)

var serviceCookie string

func init() {
	cookie := make([]byte, 10)
	_, err := rand.Read(cookie)
	if err != nil {
		log.Fatal("rand.Read: ", err)
	}
	serviceCookie = fmt.Sprintf("%x", string(cookie))
	serviceCookie = "AAA"
}

const (
	serviceBindHost = `0.0.0.0`
	serviceAPIPort  = 4891
	serviceUIPort   = 1984
)

const (
	defaultPageSize = uint(10)
)

type pullResponse struct {
	Records []capture.RecordMeta `json:"records"`
	Pages   uint                 `json:"pages"`
	Page    uint                 `json:"page"`
}

func replyCode(w http.ResponseWriter, httpCode int) {
	w.WriteHeader(httpCode)
	_, _ = w.Write([]byte(http.StatusText(httpCode)))
}

type writeOption uint8

const (
	writeNone         writeOption = 0
	writeWire                     = 1
	writeEmbed                    = 2
	writeRequestBody              = 4
	writeResponseBody             = 8
)

func replyBinary(w http.ResponseWriter, r *http.Request, record *capture.Record, opts writeOption) {
	var (
		optRequestBody  = opts&writeRequestBody > 0
		optResponseBody = opts&writeResponseBody > 0
		optWire         = opts&writeWire > 0
		optEmbed        = opts&writeEmbed > 0
	)

	if opts == writeNone {
		return
	}

	if optRequestBody && optResponseBody {
		// we should never have both options enabled at the same time.
		replyCode(w, http.StatusInternalServerError)
		return
	}

	u, err := url.Parse(record.URL)
	if err != nil {
		log.Printf("url.Parse: %w", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	basename := u.Host + "-" + path.Base(u.Path)
	basename = reUnsafeFile.ReplaceAllString(basename, "-")
	basename = strings.Trim(reRepeatedDash.ReplaceAllString(basename, "-"), "-")
	if path.Ext(basename) == "" {
		basename = basename + ".txt"
	}

	buf := bytes.NewBuffer(nil)

	if optWire {
		var headers http.Header
		if optRequestBody {
			headers = record.RequestHeader.Header
		}
		if optResponseBody {
			headers = record.Header.Header
		}
		for k, vv := range headers {
			for _, v := range vv {
				buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
			}
		}
		buf.WriteString("\r\n")
	}

	if optRequestBody || optResponseBody {

		if optRequestBody {
			buf.Write(record.RequestBody)
		}
		if optResponseBody {
			buf.Write(record.Body)
		}

		if optEmbed {
			embedContentType := "text/plain; charset=utf-8"
			w.Header().Set(
				"Content-Type",
				embedContentType,
			)
			w.Write(buf.Bytes())
		} else {
			w.Header().Set(
				"Content-Disposition",
				fmt.Sprintf(`attachment; filename="%s"`, basename),
			)
			http.ServeContent(w, r, "", record.DateEnd, bytes.NewReader(buf.Bytes()))
		}
	}

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
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf)
}

func getCaptureRecord(uuid string) (*capture.Record, error) {
	var record capture.Record

	res := storage.Find(
		db.Cond{"uuid": uuid},
	).Select(
		"uuid",
		"origin",
		"method",
		"status",
		"content_type",
		"content_length",
		"host",
		"url",
		"path",
		"scheme",
		"date_start",
		"date_end",
		"time_taken",
		"header",
		"request_header",
		db.Raw("hex(body) AS body"),
		db.Raw("hex(request_body) AS request_body"),
	)

	if err := res.One(&record); err != nil {
		return nil, err
	}

	{
		body, err := hex.DecodeString(string(record.RequestBody))
		if err != nil {
			return nil, err
		}
		record.RequestBody = body
	}

	{
		body, err := hex.DecodeString(string(record.Body))
		if err != nil {
			return nil, err
		}
		record.Body = body
	}

	return &record, nil
}

func recordMetaHandler(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")

	record, err := getCaptureRecord(uuid)
	if err != nil {
		log.Printf("getCaptureRecord: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	replyJSON(w, record.RecordMeta)
}

func recordHandler(w http.ResponseWriter, r *http.Request, opts writeOption) {
	uuid := chi.URLParam(r, "uuid")

	record, err := getCaptureRecord(uuid)
	if err != nil {
		log.Printf("getCaptureRecord: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	replyBinary(w, r, record, opts)
}

func requestContentHandler(w http.ResponseWriter, r *http.Request) {
	recordHandler(w, r, writeRequestBody)
}

func requestWireHandler(w http.ResponseWriter, r *http.Request) {
	recordHandler(w, r, writeRequestBody|writeWire)
}

func requestEmbedHandler(w http.ResponseWriter, r *http.Request) {
	recordHandler(w, r, writeRequestBody|writeEmbed)
}

func responseContentHandler(w http.ResponseWriter, r *http.Request) {
	recordHandler(w, r, writeResponseBody)
}

func responseWireHandler(w http.ResponseWriter, r *http.Request) {
	recordHandler(w, r, writeResponseBody|writeWire)
}

func responseEmbedHandler(w http.ResponseWriter, r *http.Request) {
	recordHandler(w, r, writeResponseBody|writeEmbed)
}

// capturesHandler service serves paginated requests.
func capturesHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var response pullResponse

	q := r.URL.Query().Get("q")

	q = reUnsafeChars.ReplaceAllString(q, " ")
	q = reRepeatedBlank.ReplaceAllString(q, " ")

	page := uint(1)
	{
		i, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
		if err == nil {
			page = uint(i)
		}
	}

	pageSize := defaultPageSize
	{
		i, err := strconv.ParseUint(r.URL.Query().Get("page_size"), 10, 64)
		if err == nil {
			pageSize = uint(i)
		}
	}

	// Result set
	res := storage.Find().
		Select(
			"id",
			"uuid",
			"origin",
			"method",
			"status",
			"content_type",
			"content_length",
			"host",
			"url",
			"path",
			"scheme",
			"date_start",
			"date_end",
			"time_taken",
		).
		OrderBy("id")

	if q != "" {
		terms := strings.Split(q, " ")
		conds := db.Or()

		for _, term := range terms {
			conds = conds.Or(
				db.Or(
					db.Cond{"host LIKE": "%" + term + "%"},
					db.Cond{"origin LIKE": "%" + term + "%"},
					db.Cond{"path LIKE": "%" + term + "%"},
					db.Cond{"content_type LIKE": "%" + term + "%"},
					db.Cond{"method": term},
					db.Cond{"scheme": term},
					db.Cond{"status": term},
				),
			)
		}

		res = res.Where(conds)
	}

	res = res.Paginate(pageSize).Page(page)

	// Pulling information page.
	if err = res.All(&response.Records); err != nil {
		log.Printf("res.All: %q", err)
		replyCode(w, http.StatusInternalServerError)
		return
	}

	// Getting total number of pages.
	response.Page = page
	response.Pages, _ = res.TotalPages()

	replyJSON(w, response)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			auth = r.URL.Query().Get("auth")
		}
		if auth != "" {
			chunks := strings.SplitN(auth, " ", 2)
			auth = chunks[len(chunks)-1]
			if auth == serviceCookie {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusForbidden)
	})
}

func apiServer() (string, error) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	cors := cors.New(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	r.Use(cors.Handler)

	r.Use(authMiddleware)

	r.Route("/records", func(r chi.Router) {
		r.Get("/", capturesHandler)

		r.Route("/{uuid}", func(r chi.Router) {
			r.Get("/", recordMetaHandler)

			r.Route("/request", func(r chi.Router) {
				r.Get("/", requestContentHandler)
				r.Get("/raw", requestWireHandler)
				r.Get("/embed", requestEmbedHandler)
			})

			r.Route("/response", func(r chi.Router) {
				r.Get("/", responseContentHandler)
				r.Get("/raw", responseWireHandler)
				r.Get("/embed", responseEmbedHandler)
			})
		})
	})

	r.HandleFunc("/live", liveHandler)

	addr := fmt.Sprintf("%s:%d", serviceBindHost, serviceAPIPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Serving API.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("ListenAndServe: %v", err)
			return
		}
	}()

	return addr, nil
}

func uiServer(apiAddr string) (string, error) {
	statikFS, err := fs.New()
	if err != nil {
		return "", err
	}

	addr := fmt.Sprintf("%s:%d", serviceBindHost, serviceUIPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: http.FileServer(statikFS),
	}

	// Serving API.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("ListenAndServe: %v", err)
			return
		}
	}()

	return addr, nil
}

// startServices starts an http server that provides websocket and rest
// services.
func startServices() error {

	apiAddr, err := apiServer()
	if err != nil {
		log.Fatal("error starting API server: ", err)
	}
	log.Printf("started API server at %v", apiAddr)

	uiAddr, err := uiServer(apiAddr)
	if err != nil {
		log.Fatal("error starting UI server: ", err)
	}
	log.Printf("started UI server at %v", uiAddr)

	uiAddrWithToken := fmt.Sprintf("http://%s/?source=%s&auth=%s", uiAddr, apiAddr, serviceCookie)
	if err := browser.OpenURL(uiAddrWithToken); err != nil {
		log.Printf("failed to open browser: %v", err)
	}

	log.Printf("Watch live capture at %s", uiAddrWithToken)

	return err
}
