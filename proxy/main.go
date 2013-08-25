/*
	Hyperfox - Man In The Middle Proxy for HTTP(s).

	The github.com/xiam/hyperfox/proxy provides a generic HTTP(s) proxy designed
	to execute special callbacks at specific stages of a Man In The Middle
	operation.

	Written by Carlos Reventlov <carlos@reventlov.com>
	License MIT
*/

// This package provides a generic HTTP(s) proxy that can execute callbacks at
// specific stages of a Man In The Middle operation.
//
// Callbacks have special names and meanings: Director, Interceptor, Writer and
// Logger.
//
// When a client intends to request an URL from a destination server, this
// request is first captured and may be modified by Director functions:
//
// [legitimate client request] -> [director] -> [actual request] -> [server]
//
// The server gets the proxied request and creates a response, then the
// response is intercepted by Hyperfox and passed to Interceptor functions,
// Interceptor functions can be used to modify response headers or body.
//
// [legitimate server response] -> [interceptor] -> ... -> [client]
//
// After the request has been intercepted, Writer functions are then called,
// Writer functions can be used to create io.WriteCloser functions that may
// be used to output data to disk or other medium. It is illegal to use Writer
// functions to edit Request or Response properties.
//
// [interceptor] -> [writer] -> ... -> [client]
//
// Just before sending the final response to the client, Logger functions are
// invoked. These functions may log the final response data but they must not
// edit any property.
//
// [writer] -> [logger] -> [client].
//
// Here is a final diagram on the order of proxy callbacks:
//
// [client request] -> [director] -> [server]
//
// ...server does its magic and generates a response...
//
// [server response] -> [interceptor] -> [writer] -> [logger] -> [client]
//
package proxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	// Default directory to write request data.
	Workdir = `output`
	// Default address to bind into.
	DefaultBindAddress = `0.0.0.0:9999`
)

var (
	ErrProxyRequestFailed = errors.New(`Proxy request failed: %s.`)
	ErrWriterFailed       = errors.New(`Writer function failed: %s.`)
)

// This type defines functions that are called when proxied server responses
// arrive, before sending any response data to the client:
//
// [legitimate server response] -> ... -> [writer] -> [client]
//
// Writer functions read request headers and generate a io.WriterCloser that
// is to be called when reading server response data.
//
// You can use a io.WriteCloser function to save the original content of a
// server response into a file but you may not use Writer functions to edit
// Request or Response properties.
//
type Writer func(*ProxyRequest) (io.WriteCloser, error)

// This type defines functions that are called when client requests are
// received:
//
// [legitimate client request] -> [director] -> [...] -> [destination server]
//
// You may use Director functions to modify requests headers or body before
// sending them to the destination client:
//
type Director func(*ProxyRequest) error

// This type defines functions that are called when proxied server responses
// arrive, before sending any response data to the client:
//
// [legitimate server response] -> [interceptor] -> [...] -> [client]
//
// You may use Interceptor functions to edit Response properties such as header
// or body.
//
type Interceptor func(*ProxyRequest) error

// This type defines functions that are called when proxied server responses
// arrive and immediately before sending them to the client.
//
// [legitimate server response] -> [...] -> [logger] -> [client]
//
// You may use Logger functions to view or log the data that is to be sent to
// the client but you may not use logger functions to edit any property.
//
type Logger func(*ProxyRequest) error

// Path separator.
const PS = string(os.PathSeparator)

// A general proxy definition that has an address and can intercept, log and
// modify requests.
type Proxy struct {
	// Standard HTTP server
	srv http.Server
	// Address to bind into.
	Bind string
	// Writer functions.
	Writers []Writer
	// Director functions.
	Directors []Director
	// Interceptor functions.
	Interceptors []Interceptor
	// Logger functions.
	Loggers []Logger
}

// Each requests to be proxied spawns a ProxyRequests and has a unique directory
// that can be used to write logs or other data that is associated to the
// request.
type ProxyRequest struct {
	*Proxy
	http.ResponseWriter
	*http.Request
	*http.Response
	Id       string // Request ID. Must be unique.
	FileName string
}

// Returns a new Proxy.
func New() *Proxy {
	self := &Proxy{}
	self.Bind = DefaultBindAddress
	return self
}

// Returns a new ProxyRequest.
func (self *Proxy) NewProxyRequest(wri http.ResponseWriter, req *http.Request) *ProxyRequest {

	pr := &ProxyRequest{}

	pr.Proxy = self
	pr.ResponseWriter = wri
	pr.Request = req
	pr.Id = pr.requestId()
	pr.FileName = pr.fileName()

	return pr
}

// Adds a Writer function to the Proxy.
//
// Writer functions are called in the same order
// they are added.
func (self *Proxy) AddWriter(wri Writer) {
	self.Writers = append(self.Writers, wri)
}

// Adds a Director function to the Proxy.
//
// Director functions are called in the same order
// they are added.
func (self *Proxy) AddDirector(dir Director) {
	self.Directors = append(self.Directors, dir)
}

// Adds an Interceptor function to the Proxy.
//
// Interceptor functions are called in the same order
// they are added.
func (self *Proxy) AddInterceptor(dir Interceptor) {
	self.Interceptors = append(self.Interceptors, dir)
}

// Adds a Logger function to the Proxy.
//
// Logger functions are called in the same order
// they are added.
func (self *Proxy) AddLogger(dir Logger) {
	self.Loggers = append(self.Loggers, dir)
}

// Copies headers from one http.Header to another.
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

// Catches a client request and proxies it to the
// destination server, then waits for and answer and sends it back
// to the client.
//
// Should not be called directly.
func (self *Proxy) ServeHTTP(wri http.ResponseWriter, req *http.Request) {
	var i int
	var err error
	var transport *http.Transport
	var scheme string

	pr := self.NewProxyRequest(wri, req)

	pr.Request.Header.Add("Host", pr.Request.Host)

	// Running directors.
	for i, _ = range self.Directors {
		self.Directors[i](pr)
	}

	pr.Request.Header.Add("Host", pr.Request.Host)

	// Creating a request that will be sent to the destination server.
	out := new(http.Request)

	if req.TLS == nil {
		transport = &http.Transport{}
		scheme = "http"
	} else {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		scheme = "https"
	}

	*out = *pr.Request
	out.Proto = "HTTP/1.1"
	out.ProtoMajor = 1
	out.ProtoMinor = 1
	out.Close = false

	out.URL.Scheme = scheme

	out.URL.Host = pr.Request.Host

	// Proxying client request to destination server.
	pr.Response, err = transport.RoundTrip(out)

	// Waiting for an answer.
	if err != nil {
		log.Printf(ErrProxyRequestFailed.Error(), err.Error())
	}

	// Running interceptors.
	for i, _ = range self.Interceptors {
		self.Interceptors[i](pr)
	}

	// Copying response headers to the response we are going to send to the
	// client.
	copyHeader(pr.ResponseWriter.Header(), pr.Response.Header)

	// Writing response status.
	pr.ResponseWriter.WriteHeader(pr.Response.StatusCode)

	wclosers := []io.WriteCloser{}

	// Running writers.
	for i, _ := range self.Writers {
		wcloser, err := self.Writers[i](pr)
		if wcloser != nil {
			wclosers = append(wclosers, wcloser)
		}
		if err != nil {
			log.Printf(ErrWriterFailed.Error(), err.Error())
		}
	}

	// Loggers.
	for i, _ = range self.Loggers {
		self.Loggers[i](pr)
	}

	// Writing response.
	if pr.Response.Body != nil {
		writers := []io.Writer{pr.ResponseWriter}
		for i, _ := range wclosers {
			writers = append(writers, wclosers[i])
		}
		io.Copy(io.MultiWriter(writers...), pr.Response.Body)
	}

	// Closing response.
	pr.Response.Body.Close()

	// Closing associated writers.
	for i, _ := range wclosers {
		wclosers[i].Close()
	}
}

// Returns an appropriate name for a file that needs to be associated with a
// ProxyRequest.
func (self *ProxyRequest) fileName() string {

	file := strings.Trim(self.Request.URL.Path, "/")

	if file == "" {
		file = "index"
	}

	offset := strings.LastIndex(self.Request.RemoteAddr, ":")

	file = self.Request.RemoteAddr[0:offset] + PS + self.Request.Host + PS + file

	return file
}

// Returns a unique identificator for a ProxyRequest.
func (self *ProxyRequest) requestId() string {

	t := time.Now().Local()

	// This should provide an extremely low chance to create race conditions.
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

// Starts a web server.
func (self *Proxy) Start() error {

	self.srv = http.Server{
		Addr:    self.Bind,
		Handler: self,
	}

	log.Printf("Listening for HTTP client requests at %s.\n", self.Bind)

	return self.srv.ListenAndServe()
}

// Starts a HTTPS web server.
func (self *Proxy) StartTLS(cert string, key string) error {

	self.srv = http.Server{
		Addr:    self.Bind,
		Handler: self,
	}

	log.Printf("Listening for HTTPs client request at %s.\n", self.Bind)

	return self.srv.ListenAndServeTLS(cert, key)
}
