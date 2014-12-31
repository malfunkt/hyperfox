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

// Package proxy provides methods for creating a proxy programmatically.
package proxy

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/xiam/hyperfox/util/otf"
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
	// Writer functions.
	writers []BodyWriteCloser
	// Director functions.
	directors []Director
	// Interceptor functions.
	interceptors []Interceptor
	// Logger functions.
	loggers []Logger
}

// ProxiedRequest struct provides properties for executing a *http.Request and
// proxying it into a http.ResponseWriter.
type ProxiedRequest struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Response       *http.Response
}

// NewProxy creates and returns a Proxy reference.
func NewProxy() *Proxy {
	p := new(Proxy)
	return p
}

// Reset clears the list of interfaces.
func (p *Proxy) Reset() {
	p.writers = []BodyWriteCloser{}
	p.directors = []Director{}
	p.interceptors = []Interceptor{}
	p.loggers = []Logger{}
}

// NewProxiedRequest creates and returns a ProxiedRequest reference.
func (p *Proxy) newProxiedRequest(wri http.ResponseWriter, req *http.Request) *ProxiedRequest {

	pr := &ProxiedRequest{
		ResponseWriter: wri,
		Request:        req,
	}

	return pr
}

// AddBodyWriteCloser appends a struct that satisfies the BodyWriteCloser
// interface to the list of body write closers.
func (p *Proxy) AddBodyWriteCloser(wri BodyWriteCloser) {
	p.writers = append(p.writers, wri)
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
func (p *Proxy) ServeHTTP(wri http.ResponseWriter, req *http.Request) {

	var err error
	var transport *http.Transport

	pr := p.newProxiedRequest(wri, req)

	out := new(http.Request)
	*out = *req

	if req.TLS == nil {
		transport = &http.Transport{}
		out.URL.Scheme = "http"
	} else {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		}
		out.URL.Scheme = "https"
	}

	// Making sure the Host header is present.
	out.URL.Host = out.Host
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
	io.Copy(io.MultiWriter(body, bodyCopy), out.Body)

	out.Body = ioutil.NopCloser(body)

	// Proxying client request to destination server.
	if pr.Response, err = transport.RoundTrip(out); err != nil {
		log.Printf("RoundTrip: %q", err)
		wri.WriteHeader(http.StatusInternalServerError)
		return
	}

	// (Response received).

	// Resetting body (so it can be read later)
	out.Body = ioutil.NopCloser(bodyCopy)

	// Walking over interceptos.
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

	defer pr.Response.Body.Close()

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
func (p *Proxy) Start(addr string) error {

	p.srv = http.Server{
		Addr:    addr,
		Handler: p,
	}

	log.Printf("Listening for incoming HTTP client requests on %s.\n", addr)

	if err := p.srv.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func certificateLookup(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {

	cert, key, err := otf.CreateKeyPair(clientHello.ServerName)

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
func (p *Proxy) StartTLS(addr string) error {

	p.srv = http.Server{
		Addr:    addr,
		Handler: p,
		TLSConfig: &tls.Config{
			GetCertificate:     certificateLookup,
			InsecureSkipVerify: false,
		},
	}

	log.Printf("Listening for incoming HTTPs client requests on %s.\n", addr)

	otf.SetRootCACert(os.Getenv(EnvSSLCert))
	otf.SetRootCAKey(os.Getenv(EnvSSLKey))

	if err := p.srv.ListenAndServeTLS(os.Getenv(EnvSSLCert), os.Getenv(EnvSSLKey)); err != nil {
		return err
	}

	return nil
}
