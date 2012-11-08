/*
	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package proxy

import (
	"io"
	"log"
	"net/http"
	"time"
	"os"
	"fmt"
	"strings"
)

/*
	Returns a io.WriteCloser that will be called
	everytime new content is received from the destination.

	Writer functions should not edit response headers or
	body.
*/
type Writer func(*ProxyRequest) io.WriteCloser

/*
	Called before giving any output to the client.

	Director functions can be used to edit response headers
	and body before arriving to the client.
*/
type Director func(*ProxyRequest) error

/*
	Called right before sending content to the client.

	Logger functions should not edit response headers or
	body.
*/
type Logger func(*ProxyRequest) error

/*
	Storage directories.
*/
var ArchiveDir = "archive"
var ClientDir = "client"

const PS = string(os.PathSeparator)

/*
	Proxy.
*/
type Proxy struct {
	srv       http.Server
	Bind      string
	Writers   []Writer
	Directors []Director
	Loggers   []Logger
}

type ProxyRequest struct {
	*Proxy
	http.ResponseWriter
	*http.Request
	*http.Response
	Id string
	FileName string
}

/*
	Returns a new Proxy.
*/
func New() *Proxy {
	self := &Proxy{}
	self.Writers = []Writer{}
	self.Directors = []Director{}
	self.Bind = "0.0.0.0:9999"
	return self
}

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
	destination server.

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
		writers := []io.Writer{ pr.ResponseWriter }
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
	with a response.
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
	Returns an appropriate name for a file that needs to be associated
	with a response.
*/
/*
func ArchiveFile(res *http.Response) string {

	contentType := res.Header.Get("Content-Type")

	file := strings.Trim(res.Request.URL.Path, "/")

	if file == "" {
		file = "index"
	}

	if path.Ext(file) == "" {
		file = file + "." + mimext.Ext(contentType)
	}

	if res.Header.Get("Content-Encoding") == "gzip" {
		file = file + ".gz"
	}

	file = ArchiveDir + PS + res.Request.URL.Host + PS + file

	return file
}
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
func generateRequestId(req *http.Request) string {

	t := time.Now().Local()
	name := fmt.Sprintf(
		"%04d%02d%02d-%02d%02d%02d-%09d",
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond(),
	)
	return name + ".bin"
}

/*
	Returns an appropriate name for a file that needs to be associated
	with a request.
*/
/*
func ClientFile(res *http.Response) string {

	file := strings.Trim(res.Request.URL.Path, "/")

	if file == "" {
		file = "index"
	}

	clientAddr := strings.SplitN(res.Request.RemoteAddr, ":", 2)

	file = ClientDir + PS + clientAddr[0] + PS + res.Request.URL.Host + PS + file + PS + generateRequestId(res.Request)

	return file
}

func Workdir(dir string) error {
	return os.MkdirAll(dir, os.ModeDir|os.FileMode(0755))
}
*/

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

	err := self.srv.ListenAndServe()

	if err != nil {
		log.Printf("Failed to bind.\n")
	}

	return err
}
