package proxy

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"testing"
	"time"
)

var (
	proxy    *Proxy
	sslProxy *Proxy
)

const (
	listenHTTPAddr  = `localhost.example.org:13080`
	listenHTTPsAddr = `localhost.example.org:13443`
)

type writeCloser struct {
	bytes.Buffer
	closed bool
}

func (w *writeCloser) Close() error {
	w.closed = true
	return nil
}

type testLogger struct {
	logged bool
}

func (l *testLogger) Log(pr *ProxiedRequest) error {
	l.logged = true
	return nil
}

type testWriteCloser struct {
	wc *writeCloser
}

func (w *testWriteCloser) NewWriteCloser(*http.Response) (io.WriteCloser, error) {
	w.wc = &writeCloser{}
	return w.wc, nil
}

type testInterceptor struct {
}

func (i testInterceptor) Intercept(res *http.Response) error {
	var err error
	var buf []byte

	// Forging response status.
	res.StatusCode = 500

	// Reading response.
	if buf, err = ioutil.ReadAll(res.Body); err != nil {
		return err
	}

	// Modifying response.
	buf = bytes.Replace(buf, []byte("nmap.org"), []byte("mapn.tld"), -1)

	// Replacing response body.
	res.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

	return nil
}

type testDirectorSSL struct {
}

func (d testDirectorSSL) Direct(req *http.Request) error {
	newRequest, _ := http.NewRequest("GET", "https://www.example.org/", nil)
	*req = *newRequest
	return nil
}

type testDirector struct {
}

func (d testDirector) Direct(req *http.Request) error {
	newRequest, _ := http.NewRequest("GET", "http://nmap.org/", nil)
	*req = *newRequest
	return nil
}

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
	proxy = NewProxy()

	go func(t *testing.T) {
		if err := proxy.Start(listenHTTPAddr); err != nil {
			t.Fatal(err)
		}
	}(t)

	time.Sleep(time.Millisecond * 100)
}

func TestListenHTTPs(t *testing.T) {
	sslProxy = NewProxy()

	go func(t *testing.T) {
		if err := sslProxy.StartTLS(listenHTTPsAddr); err != nil {
			t.Fatal(err)
		}
	}(t)

	time.Sleep(time.Millisecond * 100)
}

func TestProxyResponse(t *testing.T) {
	var req *http.Request
	var err error

	// Creating a request.
	if req, err = http.NewRequest("GET", "https://www.example.org", nil); err != nil {
		t.Fatal(err)
	}

	// Creating a response writer.
	wri := newTestResponseWriter()

	// Executing request.
	proxy.ServeHTTP(wri, req)

	// Verifying response.
	var size int
	if size, err = strconv.Atoi(wri.header.Get("Content-Length")); err != nil {
		t.Fatal(err)
	}

	if size <= 0 {
		t.Fatal("Expecting some content.")
	}

	if size != len(wri.buf.Bytes()) {
		t.Fatal("Content length does not match actual content.")
	}
}

func TestDirectorInterface(t *testing.T) {
	var req *http.Request
	var err error

	// Creating a request to golang.org
	if req, err = http.NewRequest("GET", "http://www.golang.org", nil); err != nil {
		t.Fatal(err)
	}

	// Creating a response writer.
	wri := newTestResponseWriter()

	// Adding a director that will change the request destination to insecure.org
	proxy.AddDirector(testDirector{})

	// Executing request.
	proxy.ServeHTTP(wri, req)

	if bytes.Count(wri.buf.Bytes(), []byte(`nmap.org`)) == 0 {
		t.Fatal("Director failed to take over the request.")
	}
}

func TestInterceptorInterface(t *testing.T) {
	var req *http.Request
	var err error

	// Creating a request to golang.org
	if req, err = http.NewRequest("GET", "http://www.golang.org", nil); err != nil {
		t.Fatal(err)
	}

	// Creating a response writer.
	wri := newTestResponseWriter()

	// Adding an interceptos that will alter the response status and some texts
	// from the original page.
	proxy.AddInterceptor(testInterceptor{})

	// Executing request.
	proxy.ServeHTTP(wri, req)

	if wri.status != 500 {
		t.Fatal("Expecting status change.")
	}

	if bytes.Count(wri.buf.Bytes(), []byte("mapn.tld")) == 0 {
		t.Fatal("Interceptor failed to modify the response.")
	}
}

func TestBodyWriteCloserInterface(t *testing.T) {
	var req *http.Request
	var err error

	// Creating a request to golang.org
	if req, err = http.NewRequest("GET", "http://www.golang.org", nil); err != nil {
		t.Fatal(err)
	}

	// Creating a response writer.
	wri := newTestResponseWriter()

	// Adding write closer that will receive all the data and then a closing
	// instruction.
	w := &testWriteCloser{}
	proxy.AddBodyWriteCloser(w)

	// Executing request.
	proxy.ServeHTTP(wri, req)

	if wri.status != 500 {
		t.Fatal("Expecting status change.")
	}

	if bytes.Count(wri.buf.Bytes(), []byte("mapn.tld")) == 0 {
		t.Fatal("Interceptor failed to modify the response.")
	}

	if bytes.Equal(w.wc.Bytes(), wri.buf.Bytes()) == false {
		t.Fatal("Buffers must be equal.")
	}

	if w.wc.closed == false {
		t.Fatal("WriteCloser must be closed.")
	}
}

func TestLoggerInterface(t *testing.T) {
	var req *http.Request
	var err error

	// Creating a request to golang.org
	if req, err = http.NewRequest("GET", "http://www.example.org", nil); err != nil {
		t.Fatal(err)
	}

	// Creating a response writer.
	wri := newTestResponseWriter()

	// Adding write closer that will receive all the data and then a closing
	// instruction.
	log := &testLogger{}
	proxy.AddLogger(log)

	// Executing request.
	proxy.ServeHTTP(wri, req)

	if log.logged == false {
		t.Fatal("Expected flag change.")
	}
}

func TestActualHTTPClient(t *testing.T) {
	// Reset lists.
	proxy.Reset()

	// Adding a director that will change the request destination to insecure.org
	proxy.AddDirector(testDirector{})

	// Adding write closer that will receive all the data and then a closing
	// instruction.
	w := &testWriteCloser{}
	proxy.AddBodyWriteCloser(w)

	client := &http.Client{}

	res, err := client.Get("http://" + listenHTTPAddr)

	if err != nil {
		t.Fatal(err)
	}

	if bytes.Count(w.wc.Bytes(), []byte("insecure.org")) < 1 {
		t.Fatal("Expecting a redirection.")
	}

	buf := bytes.NewBuffer(nil)

	if _, err := io.Copy(buf, res.Body); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(buf.Bytes(), w.wc.Bytes()) == false {
		t.Fatal("Responses differ.")
	}
}

func TestHTTPsDirectorInterface(t *testing.T) {
	sslProxy.Reset()
	// Adding a director that will change the request destination to insecure.org
	sslProxy.AddDirector(testDirectorSSL{})
	log.Printf("SSL proxy server will be open for 10 secs from now...")
	time.Sleep(time.Second * 10)
}
