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

package capture

import (
	"bytes"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/malfunkt/hyperfox/pkg/proxy"
)

const listenHTTPAddr = `127.0.0.1:37400`

var px *proxy.Proxy

type testRecordWriter struct {
	header http.Header
	buf    *bytes.Buffer
	status int
}

func (rw *testRecordWriter) Header() http.Header {
	return rw.header
}

func (rw *testRecordWriter) Write(buf []byte) (int, error) {
	return rw.buf.Write(buf)
}

func (rw *testRecordWriter) WriteHeader(i int) {
	rw.status = i
}

func newTestRecordWriter() *testRecordWriter {
	rw := &testRecordWriter{
		header: http.Header{},
		buf:    bytes.NewBuffer(nil),
	}
	return rw
}

func TestListenHTTP(t *testing.T) {
	px = proxy.NewProxy()

	go func() {
		time.Sleep(time.Millisecond * 100)
		px.Stop()
	}()

	if err := px.Start(listenHTTPAddr); err != nil {
		if !strings.Contains(err.Error(), "use of closed network connection") {
			t.Fatal(err)
		}
	}
}

func TestWriteCloser(t *testing.T) {
	var req *http.Request
	var err error

	res := make(chan *Record, 10)

	px.AddBodyWriteCloser(New(res))

	urls := []string{
		"http://golang.org/src/database/sql/",
		"http://golang.org/",
		"https://www.example.org",
		"http://nmap.org",
	}

	go func() {
		for r := range res {
			log.Printf("Captured: %s, %d, (%d bytes) - %dns", r.URL, r.Status, len(r.Body), r.TimeTaken)
		}
	}()

	for i := range urls {
		// Creating a response writer.
		wri := newTestRecordWriter()

		// Creating a request
		if req, err = http.NewRequest("GET", urls[i], nil); err != nil {
			t.Fatal(err)
		}

		// Executing request.
		px.ServeHTTP(wri, req)
	}

	time.Sleep(time.Second * 1)
}
