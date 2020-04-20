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
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/malfunkt/hyperfox/pkg/plugins/capture"
	_ "github.com/malfunkt/hyperfox/ui/statik"
	"upper.io/db.v3"
)

var (
	reUnsafeChars   = regexp.MustCompile(`[^0-9a-zA-Z\s\.]`)
	reUnsafeFile    = regexp.MustCompile(`[^0-9a-zA-Z-_]`)
	reRepeatedDash  = regexp.MustCompile(`-+`)
	reRepeatedBlank = regexp.MustCompile(`\s+`)
)

type writeOption uint8

const (
	writeNone         writeOption = 0
	writeWire         writeOption = 1
	writeEmbed        writeOption = 2
	writeRequestBody  writeOption = 4
	writeResponseBody writeOption = 8
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
		replyCode(w, http.StatusInternalServerError)
		return
	}

	basename := u.Host + "-" + path.Base(u.Path)
	basename = reUnsafeFile.ReplaceAllString(basename, "-")
	basename = strings.Trim(reRepeatedDash.ReplaceAllString(basename, "-"), "-")
	if optWire {
		basename = basename + "-raw"
	}

	ext := path.Ext(u.Path)
	if ext == "" {
		ext = ".txt"
	}
	basename = basename + ext

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
		var header http.Header

		var bodyContentType string
		if optRequestBody {
			header = record.RequestHeader.Header
			buf.Write(record.RequestBody)
		}
		if optResponseBody {
			header = record.Header.Header
			buf.Write(record.Body)
		}
		bodyContentType = header.Get("Content-Type")

		if optEmbed {
			embedContentType := "text/plain; charset=utf-8"
			if strings.HasPrefix(bodyContentType, "image/") {
				embedContentType = bodyContentType
			}
			w.Header().Set(
				"Content-Type",
				embedContentType,
			)

			out := bytes.NewBuffer(nil)
			tee := io.TeeReader(buf, out)

			gz, err := gzip.NewReader(tee)
			if err == nil {
				dst := bytes.NewBuffer(nil)
				_, _ = io.Copy(dst, gz)
				out = dst
			} else {
				_, _ = ioutil.ReadAll(tee)
			}

			_, err = w.Write(out.Bytes())
			if err != nil {
				log.Printf("failed to send raw text: %v", err)
			}
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
		conds := db.And()

		for _, term := range terms {
			conds = conds.And(
				db.Or(
					db.Cond{"keywords LIKE": "%" + term + "%"},
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
