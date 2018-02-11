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

// Package proxy provides methods for creating a proxy programmatically.
package proxy

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/malfunkt/hyperfox/lib/gencert"
	"github.com/tv42/httpunix"
)

const (
	// EnvSSLKey defines the name for the environment variable that holds the
	// root SSL key.
	EnvSSLKey = `HYPERFOX_SSL_KEY`
	// EnvSSLCert defines the name for the environment variable that holds the
	// root SSL certificate..
	EnvSSLCert = `HYPERFOX_SSL_CERT`
)

// BodyWriteCloser interface returns a io.WriteCloser where a copy of the
// response body will be written. The io.WriteCloser's Close() method will be
// called after the body has been written entirely.
//
// destination -> ... -> BodyWriteCloser -> client -> ...
type BodyWriteCloser interface {
	NewWriteCloser(*http.Response) (io.WriteCloser, error)
}

// Director interface gets a reference of the http.Request sent by an user
// before sending it to the destination. The Direct() method may modify the
// client's request.
//
// client -> Director -> destination
type Director interface {
	Direct(*http.Request) error
}

// Interceptor interface gets a reference of the http.Response sent by the
// destination before arriving to the client. The Interceptor() method may
// modify the destination's response.
//
// destination -> Interceptor -> ... -> client -> ...
type Interceptor interface {
	Intercept(*http.Response) error
}

// Logger interface gets a reference of the ProxiedRequest after the response
// has been writte to the client.
//
// The Logger() method must not modify any *ProxiedRequest properties.
//
// destination -> ... -> client -> Logger
type Logger interface {
	Log(*ProxiedRequest) error
}

// Proxy struct provides methods and properties for creating a proxy
// programatically.
type Proxy struct {
	// Standard HTTP server
	srv http.Server
	// RoundTrip to proxied service
	rt http.RoundTripper
	// Writer functions.
	writers []BodyWriteCloser
	// Director functions.
	directors []Director
	// Interceptor functions.
	interceptors []Interceptor
	// Logger functions.
	loggers []Logger
	// The proxy's listen address
	listenAddr string
	// Destination server, if not empty, the proxy acts as a reverse proxy
	dstAddr string
}

// ProxiedRequest struct provides properties for executing a *http.Request and
// proxying it into a http.ResponseWriter.
type ProxiedRequest struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Response       *http.Response
}

// NewProxy creates and returns a Proxy reference. the proxy will listen on the
// given address. if dstAddr is not empty, the proxy will be a reverse proxy,
// and forward request data to dstAddr
func NewProxy(listenAddr, dstServer string) *Proxy {
	return &Proxy{
		listenAddr: listenAddr,
		dstAddr:    dstServer,
	}
}

// Reset clears the list of interfaces.
func (p *Proxy) Reset() {
	p.writers = []BodyWriteCloser{}
	p.directors = []Director{}
	p.interceptors = []Interceptor{}
	p.loggers = []Logger{}
}

// NewProxiedRequest creates and returns a ProxiedRequest reference.
func (p *Proxy) newProxiedRequest(w http.ResponseWriter, r *http.Request) *ProxiedRequest {
	return &ProxiedRequest{
		ResponseWriter: w,
		Request:        r,
	}
}

// AddBodyWriteCloser appends a struct that satisfies the BodyWriteCloser
// interface to the list of body write closers.
func (p *Proxy) AddBodyWriteCloser(wc BodyWriteCloser) {
	p.writers = append(p.writers, wc)
}

// AddDirector appends a struct that satisfies the Director interface to the
// list of directors.
func (p *Proxy) AddDirector(dir Director) {
	p.directors = append(p.directors, dir)
}

// AddInterceptor appends a struct that satisfies the Interceptor interface to
// the list of interceptors.
func (p *Proxy) AddInterceptor(dir Interceptor) {
	p.interceptors = append(p.interceptors, dir)
}

// AddLogger appends a struct that satisfies the Logger interface to the list
// of loggers.
func (p *Proxy) AddLogger(dir Logger) {
	p.loggers = append(p.loggers, dir)
}

// copyHeader copies headers from one http.Header to another.
// http://golang.org/src/pkg/net/http/httputil/reverseproxy.go#L72
func copyHeader(dst http.Header, src http.Header) {
	for k := range dst {
		dst.Del(k)
	}
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// ServeHTTP catches a client request and proxies it to the destination server,
// then waits for the server's answer and sends it back to the client.
//
// (this method should not be called directly).
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pr := p.newProxiedRequest(w, r)

	out := new(http.Request)
	// Copy request.
	*out = *r

	if r.TLS == nil {
		out.URL.Scheme = "http"
	} else {
		out.URL.Scheme = "https"
	}

	if p.dstAddr != "" {
		out.Host = p.dstAddr
	}

	// Making sure the Host header is present.
	out.URL.Host = out.Host
	log.Println("Host: ", out.Host);
	out.Header.Add("Host", out.Host)

	out.Proto = "HTTP/1.1"
	out.ProtoMajor = 1
	out.ProtoMinor = 1
	out.Close = false

	// Walking over directors.
	for i := range p.directors {
		if err := p.directors[i].Direct(out); err != nil {
			log.Printf("Director: %q", err)
		}
	}

	// Intercepting request body.
	body := bytes.NewBuffer(nil)
	bodyCopy := bytes.NewBuffer(nil)

	if out.Body != nil {
		io.Copy(io.MultiWriter(body, bodyCopy), out.Body)
		out.Body.Close()
		out.Body = ioutil.NopCloser(body)
	}

	// Proxying client request to destination server.
	var err error
	if pr.Response, err = p.rt.RoundTrip(out); err != nil {
		log.Printf("RoundTrip: %q", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer pr.Response.Body.Close()

	// (Response received).

	// Resetting body (so it can be read later)
	if out.Body != nil {
		out.Body = ioutil.NopCloser(bodyCopy)
	}

	// Walking over interceptors.
	for i := range p.interceptors {
		if err := p.interceptors[i].Intercept(pr.Response); err != nil {
			log.Printf("Interceptor: %q", err)
		}
	}

	// Copying response headers to the writer we are going to send to the client.
	copyHeader(pr.ResponseWriter.Header(), pr.Response.Header)

	// Copying response status.
	pr.ResponseWriter.WriteHeader(pr.Response.StatusCode)

	// Running writers.
	ws := make([]io.WriteCloser, 0, len(p.writers))

	for i := range p.writers {
		var w io.WriteCloser
		var err error
		if w, err = p.writers[i].NewWriteCloser(pr.Response); err != nil {
			log.Printf("WriteCloser: %q", err)
			continue
		}
		ws = append(ws, w)
	}

	// Writing response.
	writers := make([]io.Writer, 0, len(ws)+1)
	writers = append(writers, pr.ResponseWriter)

	for i := range ws {
		writers = append(writers, ws[i])
	}

	if _, err := io.Copy(io.MultiWriter(writers...), pr.Response.Body); err != nil {
		log.Printf("io.Copy: %q", err)
	}

	// Closing write closers.
	for i := range ws {
		if err := ws[i].Close(); err != nil {
			log.Printf("WriteCloser.Close: %q", err)
		}
	}

	// Walking over loggers.
	for i := range p.loggers {
		if err := p.loggers[i].Log(pr); err != nil {
			log.Printf("Log: %q", err)
		}
	}
}

// Start creates an HTTP proxy server that listens on the given address.
func (p *Proxy) Start() error {
	p.srv = http.Server{
		Addr:    p.listenAddr,
		Handler: p,
	}
	p.rt = &http.Transport{}

	log.Printf("Listening for incoming HTTP client requests on %s.\n", p.listenAddr)
	if err := p.srv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func certificateLookup(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cert, key, err := gencert.CreateKeyPair(clientHello.ServerName)
	if err != nil {
		return nil, err
	}

	var tlsCert tls.Certificate
	if tlsCert, err = tls.LoadX509KeyPair(cert, key); err != nil {
		return nil, err
	}

	return &tlsCert, nil
}

// StartTLS creates an HTTPs proxy server that listens on the given address.
func (p *Proxy) StartTLS() error {
	p.srv = http.Server{
		Addr:    p.listenAddr,
		Handler: p,
		TLSConfig: &tls.Config{
			GetCertificate:     certificateLookup,
			InsecureSkipVerify: true,
		},
	}
	p.rt = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	log.Printf("Listening for incoming HTTPs client requests on %s.\n", p.listenAddr)

	gencert.SetRootCACert(os.Getenv(EnvSSLCert))
	gencert.SetRootCAKey(os.Getenv(EnvSSLKey))

	if err := p.srv.ListenAndServeTLS(os.Getenv(EnvSSLCert), os.Getenv(EnvSSLKey)); err != nil {
		return err
	}
	return nil
}

// StartUnix creates an HTTP proxy server that listens on the proxy unix socket and forwards to proxied unix socket.
func (p *Proxy) StartUnix() error {
	p.srv = http.Server{
		Addr:    "http+unix://proxy",
		Handler: p,
	}
	u := &httpunix.Transport{}
	u.RegisterLocation("proxied", p.dstAddr)
	p.rt = u
	p.AddDirector(UnixDirector{"http+unix://proxied"})

	log.Printf("Listening for incoming HTTP client requests on %s\n", p.listenAddr)

	os.Remove(p.listenAddr)
	l, err := net.Listen("unix", p.listenAddr)
	if err != nil {
		return err
	}
	defer l.Close()
	defer os.Remove(p.listenAddr)
	return p.srv.Serve(l)
}

type UnixDirector struct {
	URL string
}

func (d UnixDirector) Direct(req *http.Request) (err error) {
	req.URL, err = url.Parse(d.URL + req.RequestURI)
	return
}
