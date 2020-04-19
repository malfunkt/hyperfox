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
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	_ "github.com/malfunkt/hyperfox/ui/statik"
	"github.com/mdp/qrterminal/v3"
	"github.com/pkg/browser"
	"github.com/rakyll/statik/fs"
)

const (
	defaultUIPort  = 1984
	defaultAPIPort = 4891
)

var (
	flagUI             = flag.Bool("ui", false, "Enable UI")
	flagAPI            = flag.Bool("api", false, "Enable API (enabled automatically if --ui is provided)")
	flagUIAddr         = flag.String("ui-addr", fmt.Sprintf("%s:%d", defaultAddress, defaultUIPort), "UI server address.")
	flagAPIAddr        = flag.String("api-addr", fmt.Sprintf("%s:%d", defaultAddress, defaultAPIPort), "API server address.")
	flagDisableAPIAuth = flag.Bool("disable-api-auth", false, "Disable API server authentication.")
)

type catchAllFS struct {
	fs http.FileSystem
}

func (ca catchAllFS) Open(name string) (http.File, error) {
	file, err := ca.fs.Open(name)
	if err != nil {
		// fallback to index.html
		return ca.fs.Open("/index.html")
	}
	return file, err
}

var apiAuthToken string

func init() {
	cookie := make([]byte, 8)
	_, err := rand.Read(cookie)
	if err != nil {
		log.Fatal("rand.Read: ", err)
	}
	apiAuthToken = fmt.Sprintf("%x", string(cookie))

	// Disable debugging messages when unable to open a browser window.
	browser.Stdout = nil
	browser.Stderr = nil
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
			if auth == apiAuthToken {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusForbidden)
	})
}

func apiServer() (string, error) {
	r := chi.NewRouter()
	//r.Use(middleware.Logger)

	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           0,
	})
	r.Use(cors.Handler)

	if !*flagDisableAPIAuth {
		r.Use(authMiddleware)
	}

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

	host, port, err := net.SplitHostPort(*flagAPIAddr)
	if err != nil {
		return "", err
	}

	if host == "" {
		host = defaultAddress
	}
	if port == "" {
		port = fmt.Sprintf("%d", defaultAPIPort)
	}

	apiAddr := fmt.Sprintf("%s:%s", host, port)

	srv := &http.Server{
		Addr:    apiAddr,
		Handler: r,
	}

	// Serving API.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("ListenAndServe: %v", err)
			return
		}
	}()

	return apiAddr, nil
}

func uiServer(apiAddr string) (string, error) {

	statikFS, err := fs.New()
	if err != nil {
		return "", err
	}

	host, port, err := net.SplitHostPort(*flagUIAddr)
	if err != nil {
		return "", err
	}

	if host == "" {
		host = defaultAddress
	}
	if port == "" {
		port = fmt.Sprintf("%d", defaultAPIPort)
	}

	uiAddr := fmt.Sprintf("%s:%s", host, port)

	srv := &http.Server{
		Addr:    uiAddr,
		Handler: http.FileServer(&catchAllFS{statikFS}),
	}

	// Serving API.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("ListenAndServe: %v", err)
			return
		}
	}()

	return uiAddr, nil
}

func localAddr() (string, error) {
	conn, err := net.Dial("udp4", "1.1.1.1:53")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

func displayQRCode(uiAddr, apiAddr string) error {
	addr, err := localAddr()
	if err != nil {
		return err
	}

	_, uiPort, _ := net.SplitHostPort(uiAddr)
	_, apiPort, _ := net.SplitHostPort(apiAddr)

	addrWithToken := fmt.Sprintf("http://%s:%s/?source=%s:%s&auth=%s",
		addr,
		uiPort,
		addr,
		apiPort,
		apiAuthToken,
	)
	fmt.Println("")
	log.Printf("Open Hyperfox UI on your mobile device:")
	qrterminal.GenerateHalfBlock(addrWithToken, qrterminal.H, os.Stdout)
	return nil
}

// startServices starts an http server that provides websocket and rest
// services.
func startServices() error {
	var apiAddr string

	if *flagUI || *flagAPI {
		var err error
		apiAddr, err = apiServer()
		if err != nil {
			log.Fatal("Error starting API server: ", err)
		}
		if *flagDisableAPIAuth {
			log.Printf("Started API server at %v (auth disabled)", apiAddr)
		} else {
			log.Printf("Started API server at %v (auth token: %q)", apiAddr, apiAuthToken)
		}
	}

	if !*flagUI {
		return nil
	}

	uiAddr, err := uiServer(apiAddr)
	if err != nil {
		log.Fatal("Error starting UI server: ", err)
	}
	log.Printf("Started UI server at %v", uiAddr)

	uiAddrWithToken := fmt.Sprintf("http://%s/?source=%s&auth=%s", uiAddr, apiAddr, apiAuthToken)
	if err := browser.OpenURL(uiAddrWithToken); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}

	fmt.Println("")

	log.Printf("Watch live capture at %s", uiAddrWithToken)

	host, _, _ := net.SplitHostPort(uiAddr)
	if host != "127.0.0.1" {
		if err := displayQRCode(uiAddr, apiAddr); err != nil {
			log.Printf("Failed to display QR code: %v", err)
		}
	}

	return nil
}
