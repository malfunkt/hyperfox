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

package capture

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/xiam/hyperfox/proxy"
)

const listenHTTPAddr = `127.0.0.1:37400`

var px *proxy.Proxy

type testResponseWriter struct {
	header http.Header
	buf    *bytes.Buffer
	status int
}

func (rw *testResponseWriter) Header() http.Header {
	return rw.header
}

func (rw *testResponseWriter) Write(buf []byte) (int, error) {
	return rw.buf.Write(buf)
}

func (rw *testResponseWriter) WriteHeader(i int) {
	rw.status = i
}

func newTestResponseWriter() *testResponseWriter {
	rw := &testResponseWriter{
		header: http.Header{},
		buf:    bytes.NewBuffer(nil),
	}
	return rw
}

func TestListenHTTP(t *testing.T) {
	px = proxy.NewProxy()

	go func(t *testing.T) {
		if err := px.Start(listenHTTPAddr); err != nil {
			t.Fatal(err)
		}
	}(t)

	time.Sleep(time.Millisecond * 100)
}

func TestWriteCloser(t *testing.T) {
	var req *http.Request
	var err error

	px.AddBodyWriteCloser(Capture{})

	urls := []string{
		"http://golang.org/src/database/sql/",
		"http://golang.org/",
		"https://www.example.org",
		"http://play.golang.org/p/-URiXol0GB",
	}

	for i := range urls {

		// Creating a response writer.
		wri := newTestResponseWriter()

		// Creating a request
		if req, err = http.NewRequest("GET", urls[i], nil); err != nil {
			t.Fatal(err)
		}

		// Executing request.
		px.ServeHTTP(wri, req)
	}
}
