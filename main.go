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
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/malfunkt/hyperfox/pkg/plugins/capture"
	"github.com/malfunkt/hyperfox/pkg/plugins/logger"
	"github.com/malfunkt/hyperfox/pkg/proxy"
	"upper.io/db.v3"
)

const Version = "2.0.0"

const (
	defaultAddress = `0.0.0.0`
	defaultPort    = uint(1080)
	defaultSSLPort = uint(10443)
)

var (
	flagDatabase    = flag.String("database", "", "Path to SQLite database.")
	flagAddress     = flag.String("proxy-addr", defaultAddress, "IP address of the proxy.")
	flagPort        = flag.Uint("proxy-http-port", defaultPort, "Port to bind to (plaintext mode).")
	flagSSLPort     = flag.Uint("proxy-tls-port", defaultSSLPort, "Port to bind to (TLS mode). Requires --root-ca-cert and --root-ca-key.")
	flagSSLCertFile = flag.String("root-ca-cert", "", "Path to root CA certificate.")
	flagSSLKeyFile  = flag.String("root-ca-key", "", "Path to root CA key.")
)

var (
	sess    db.Database
	storage db.Collection
)

func main() {
	log.Printf("Hyperfox v%s // https://hyperfox.org\n", Version)
	log.Printf("By José Nieto.\n\n")

	// Parsing command line flags.
	flag.Parse()

	// Opening database.
	var err error
	sess, err = initDB()
	if err != nil {
		log.Fatal("Failed to setup database: ", err)
	}
	defer sess.Close()

	storage = sess.Collection(defaultCaptureCollection)
	if !storage.Exists() {
		log.Fatalf("No such table %q", defaultCaptureCollection)
	}

	// Is SSL enabled?
	var sslEnabled bool
	if *flagSSLPort > 0 && *flagSSLCertFile != "" {
		sslEnabled = true
	}

	// User requested SSL mode.
	if sslEnabled {
		if *flagSSLCertFile == "" {
			flag.Usage()
			log.Fatal("Missing root CA certificate")
		}

		if *flagSSLKeyFile == "" {
			flag.Usage()
			log.Fatal("Missing root CA private key")
		}

		os.Setenv(proxy.EnvSSLCert, *flagSSLCertFile)
		os.Setenv(proxy.EnvSSLKey, *flagSSLKeyFile)
	}

	// Creating proxy.
	p := proxy.NewProxy()

	// Attaching logger.
	p.AddLogger(logger.Stdout{})

	// Attaching capture tool.
	res := make(chan *capture.Record, 256)

	p.AddBodyWriteCloser(capture.New(res))

	// Saving captured data with a goroutine.
	go func(res chan *capture.Record) {
		for r := range res {
			go func(r *capture.Record) {
				err := storage.InsertReturning(r)
				if err != nil {
					log.Printf("Failed to save to database: %s", err)
				}
				message := struct {
					LastRecordID uint64 `json:"last_record_id"`
				}{r.RecordMeta.ID}
				if err := wsBroadcast(message); err != nil {
					log.Print("wsBroadcast: ", err)
				}
			}(r)
		}
	}(res)

	if !*flagDisableService {
		if err = startServices(); err != nil {
			log.Fatal("ui.Serve: ", err)
		}
	}

	fmt.Println("")

	var wg sync.WaitGroup

	// Starting proxy servers.
	if *flagPort > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.Start(fmt.Sprintf("%s:%d", *flagAddress, *flagPort)); err != nil {
				log.Fatal("Failed to bind to %s:%d (HTTP): ", *flagAddress, *flagPort, err)
			}
		}()
	}

	if sslEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.StartTLS(fmt.Sprintf("%s:%d", *flagAddress, *flagSSLPort)); err != nil {
				log.Fatal("Failed to bind to %s:%d (HTTPs): ", *flagAddress, *flagSSLPort, err)
			}
		}()
	}

	wg.Wait()
}
