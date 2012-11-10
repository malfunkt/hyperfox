/*
	Hyperfox

	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

/*
	Returns a io.WriteCloser that will be called
	everytime new content is received from the destination.

	Writer functions should not edit Response nor Request.
*/
type Writer func(*ProxyRequest) io.WriteCloser

/*
	Director functions can be used to edit request headers
	and body before sending them to the destination server.

	Director functions should not edit Response nor ResponseWriter.
*/
type Director func(*ProxyRequest) error

/*
	Interceptor functions can be used to edit response headers
	and body before arriving to the client.

	Interceptor functions should not edit ResponseWriter nor Request.
*/
type Interceptor func(*ProxyRequest) error

/*
	Logger functions could be used to log server responses.

	If you need to log client request, a Director function is more appropriate.

	Logger functions should not edit any property.
*/
type Logger func(*ProxyRequest) error

/*
	Directory to write/read files.
*/
var Workdir = "archive"

/*
	Path separator.
*/
const PS = string(os.PathSeparator)

/*
	Proxy handles ProxyRequests.
*/
type Proxy struct {
	srv          http.Server
	Bind         string
	Writers      []Writer
	Directors    []Director
	Interceptors []Interceptor
	Loggers      []Logger
}

/*
	ProxyRequest handles communication between client and server.
*/
type ProxyRequest struct {
	*Proxy
	http.ResponseWriter
	*http.Request
	*http.Response
	Id       string
	FileName string
}

/*
	Returns a new Proxy.
*/
func New() *Proxy {
	self := &Proxy{}
	self.Bind = "0.0.0.0:9999"
	return self
}

/*
	Returns a new ProxyRequest.
*/
func (self *Proxy) NewProxyRequest(wri http.ResponseWriter, req *http.Request) *ProxyRequest {

	pr := &ProxyRequest{}

	pr.Proxy = self
	pr.ResponseWriter = wri
	pr.Request = req
	pr.Id = pr.requestId()
	pr.FileName = pr.fileName()

	return pr
}

/*
	Adds a Writer function to the Proxy.

	Writer functions are called in the same order
	they are added.
*/
func (self *Proxy) AddWriter(wri Writer) {
	self.Writers = append(self.Writers, wri)
}

/*
	Adds a Director function to the Proxy.

	Director functions are called in the same order
	they are added.
*/
func (self *Proxy) AddDirector(dir Director) {
	self.Directors = append(self.Directors, dir)
}

/*
	Adds an Interceptor function to the Proxy.

	Interceptor functions are called in the same order
	they are added.
*/
func (self *Proxy) AddInterceptor(dir Interceptor) {
	self.Interceptors = append(self.Interceptors, dir)
}

/*
	Adds a Logger function to the Proxy.

	Logger functions are called in the same order
	they are added.
*/
func (self *Proxy) AddLogger(dir Logger) {
	self.Loggers = append(self.Loggers, dir)
}

// http://golang.org/src/pkg/net/http/httputil/reverseproxy.go#L72
func copyHeader(dst http.Header, src http.Header) {
	for k, _ := range dst {
		dst.Del(k)
	}
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

/*
	Catches a client request and proxies it to the
	destination server, then waits for and answer and sends it back
	to the client.

	Should not be called directly.
*/
func (self *Proxy) ServeHTTP(wri http.ResponseWriter, req *http.Request) {
	var i int
	var err error

	/* Creating a *ProxyRequest */
	pr := self.NewProxyRequest(wri, req)

	/* Applying directors before sending request. */
	for i, _ = range self.Directors {
		self.Directors[i](pr)
	}

	/* Creating a request */
	out := new(http.Request)

	transport := http.DefaultTransport

	*out = *pr.Request
	out.Proto = "HTTP/1.1"
	out.ProtoMajor = 1
	out.ProtoMinor = 1
	out.Close = false

	out.URL.Scheme = "http"
	out.URL.Host = pr.Request.Host

	out.Header.Add("Host", pr.Request.Host)

	/* Sending request */
	pr.Response, err = transport.RoundTrip(out)

	/* Waiting for an answer... */
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return
	}

	for i, _ = range self.Interceptors {
		self.Interceptors[i](pr)
	}

	/* Copying headers. */
	copyHeader(pr.ResponseWriter.Header(), pr.Response.Header)

	/* Writing status. */
	pr.ResponseWriter.WriteHeader(pr.Response.StatusCode)

	wclosers := []io.WriteCloser{}

	/* Handling writers. */
	for i, _ := range self.Writers {
		wcloser := self.Writers[i](pr)
		if wcloser != nil {
			wclosers = append(wclosers, wcloser)
		}
	}

	/* Applying loggers */
	for i, _ = range self.Loggers {
		self.Loggers[i](pr)
	}

	/* Writing response. */
	if pr.Response.Body != nil {
		writers := []io.Writer{pr.ResponseWriter}
		for i, _ := range wclosers {
			writers = append(writers, wclosers[i])
		}
		io.Copy(io.MultiWriter(writers...), pr.Response.Body)
	}

	/* Closing */
	pr.Response.Body.Close()

	for i, _ := range wclosers {
		wclosers[i].Close()
	}
}

/*
	Returns an appropriate name for a file that needs to be associated
	with a ProxyRequest.
*/
func (self *ProxyRequest) fileName() string {

	file := strings.Trim(self.Request.URL.Path, "/")

	if file == "" {
		file = "index"
	}

	addr := strings.SplitN(self.Request.RemoteAddr, ":", 2)

	file = addr[0] + PS + self.Request.Host + PS + file

	return file
}

/*
	Returns a unique identificator for a ProxyRequest.
*/
func (self *ProxyRequest) requestId() string {

	t := time.Now().Local()

	name := fmt.Sprintf(
		"%s-%04d%02d%02d-%02d%02d%02d-%09d",
		self.Request.Method,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond(),
	)

	return name
}

/*
	Starts a web server.
*/
func (self *Proxy) Start() error {

	self.srv = http.Server{
		Addr:    self.Bind,
		Handler: self,
	}

	log.Printf("Hyperfox is ready.\n")
	log.Printf("Listening at %s.\n", self.Bind)

	return self.srv.ListenAndServe()
}

/*
	Starts a HTTPS web server.
*/
func (self *Proxy) StartTLS(cert string, key string) error {

	self.srv = http.Server{
		Addr:    self.Bind,
		Handler: self,
	}

	log.Printf("Hyperfox is ready.\n")
	log.Printf("Listening (HTTPS) at %s.\n", self.Bind)

	return self.srv.ListenAndServeTLS(cert, key)
}
