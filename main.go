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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/malfunkt/hyperfox/lib/plugins/capture"
	"github.com/malfunkt/hyperfox/lib/plugins/logger"
	"github.com/malfunkt/hyperfox/lib/proxy"
	"upper.io/db.v3"
	"strings"
)

const version = "1.9.8"

const (
	defaultAddress  = `0.0.0.0`
	defaultPort     = uint(1080)
	defaultSSLPort  = uint(0)
	proxyUnixSocket = `/tmp/hyperfox`
)

var (
	flagDatabase    = flag.String("d", "", "Path to database.")
	flagAddress     = flag.String("l", defaultAddress, "Bind address.")
	flagPort        = flag.Uint("p", defaultPort, "Port to bind to.")
	flagSSLPort     = flag.Uint("s", defaultSSLPort, "Port to bind to (SSL mode).")
	flagSSLCertFile = flag.String("c", "", "Path to root CA certificate.")
	flagSSLKeyFile  = flag.String("k", "", "Path to root CA key.")
	flagUnixSocket  = flag.String("S", "", "Path to socket.")
	flagHTTPProxy   = flag.String("httpProxy", "",
		"start http reverse proxy, format example: 10.0.0.2:80-192.168.0.2:80, " +
			"part before dash is listen address, part after dash is destination address.")
	flagHTTPSProxy  = flag.String("httpsProxy", "",
		"start https reverse proxy, format example: 10.0.0.2:443-192.168.0.2:443, " +
			"part before dash is listen address, part after dash is destination address.")
)

var (
	sess    db.Database
	storage db.Collection
)

func main() {
	// Banner.
	log.Printf("Hyperfox v%s // https://hyperfox.org", version)
	log.Printf("By José Carlos Nieto.\n\n")

	// Parsing command line flags.
	flag.Parse()

	// Opening database.
	var err error
	sess, err = dbInit()
	if err != nil {
		log.Fatal("Failed to setup database: ", err)
	}
	defer sess.Close()

	storage = sess.Collection(defaultCaptureCollection)
	if !storage.Exists() {
		log.Fatal("Storage table does not exist")
	}

	// Is SSL enabled?
	if *flagSSLPort > 0 || *flagHTTPSProxy != "" {
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

	// Attaching capture tool.
	res := make(chan capture.Response, 256)
	capt := capture.New(res)

	// Saving captured data with a goroutine.
	go func() {
		for {
			select {
			case r := <-res:
				go func() {
					if _, err := storage.Insert(r); err != nil {
						log.Printf("Failed to save to database: %s", err)
					}
				}()
			}
		}
	}()

	if err = startServices(); err != nil {
		log.Fatal("startServices:", err)
	}

	fmt.Println("")

	var wg sync.WaitGroup

	// Starting proxy servers.
	if *flagPort > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := fmt.Sprintf("%s:%d", *flagAddress, *flagPort)
			if err := startHTTPProxy(addr, "", capt); err != nil {
				log.Fatalf("Failed to route mode http proxy, bind address: %s, error: %s", addr, err)
			}
		}()
	}

	// start http reverse proxy
	if *flagHTTPProxy != "" {
		addrs := strings.Split(*flagHTTPProxy, "-")
		if len(addrs) != 2 {
			log.Fatalf("'%s' is not valid http proxy address", flagHTTPProxy)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := startHTTPProxy(addrs[0], addrs[1], capt); err != nil {
				log.Fatalf("Failed to http reverse proxy, address: %s, error: %s", *flagHTTPProxy, err)
			}
		}()
	}

	if *flagSSLPort > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := fmt.Sprintf("%s:%d", *flagAddress, *flagSSLPort)
			if err := startHTTPSProxy(addr, "", capt); err != nil {
				log.Fatalf("Failed to route mode https proxy, bind address: %s, error: %s", addr, err)
			}
		}()
	}

	if *flagHTTPSProxy != "" {
		addrs := strings.Split(*flagHTTPSProxy, "-")
		if len(addrs) != 2 {
			log.Fatalf("'%s' is not valid https proxy address", *flagHTTPSProxy)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := startHTTPSProxy(addrs[0], addrs[1], capt); err != nil {
				log.Fatalf("Failed to https reverse proxy, address: %s, error: %s", *flagHTTPSProxy, err)
			}
		}()
	}

	if *flagUnixSocket != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := startUnixSocketProxy(proxyUnixSocket, *flagUnixSocket, capt); err != nil {
				log.Fatalf("Failed to bind on %s: %s", proxyUnixSocket, err)
			}
		}()
	}

	wg.Wait()
}

func startHTTPProxy(listenAddr, dstAddr string, capt *capture.Capture) error {
	p := proxy.NewProxy(listenAddr, dstAddr)
	// Attaching logger.
	p.AddLogger(logger.Stdout{})
	p.AddBodyWriteCloser(capt)
	return p.Start()
}

func startHTTPSProxy(listenAddr, dstAddr string, capt *capture.Capture) error {
	p := proxy.NewProxy(listenAddr, dstAddr)
	// Attaching logger.
	p.AddLogger(logger.Stdout{})
	p.AddBodyWriteCloser(capt)
	return p.StartTLS()
}

func startUnixSocketProxy(listenAddr, dstAddr string, capt *capture.Capture) error {
	p := proxy.NewProxy(listenAddr, dstAddr)
	// Attaching logger.
	p.AddLogger(logger.Stdout{})
	p.AddBodyWriteCloser(capt)
	return p.StartUnix()
}