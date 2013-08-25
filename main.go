/*
	Hyperfox - Man In The Middle Proxy for HTTP(s).

	This is the source of the hyperfox tool (that uses hyperfox's libraries).

	Written by Carlos Reventlov <carlos@reventlov.com>
	License MIT
*/

package main

import (
	"errors"
	"flag"
	"github.com/xiam/hyperfox/proxy"
	"github.com/xiam/hyperfox/tools/inject"
	"github.com/xiam/hyperfox/tools/intercept"
	"github.com/xiam/hyperfox/tools/logger"
	"github.com/xiam/hyperfox/tools/save"
	"log"
	"os"
)

var (
	flagListen      = flag.String("l", "0.0.0.0:9999", "Listen on [interface-address]:[port]. Example 192.168.2.33:9999.")
	flagHTTPs       = flag.Bool("s", false, "Enable HTTPs.")
	flagSSLCertFile = flag.String("c", "", "Path to SSL certificate file.")
	flagSSLKeyFile  = flag.String("k", "", "Path to SSL key file.")
	flagWorkdir     = flag.String("o", "capture", "Working directory.")
)

var (
	ErrMissingSSLCert = errors.New(`Missing SSL certificate.`)
	ErrMissingSSLKey  = errors.New(`Missing SSL certificate key.`)
	ErrBindFailed     = errors.New(`Failed to bind on the given interface: %s.`)
)

// Parses flags and initializes Hyperfox tool.
func main() {
	var err error

	flag.Parse()

	if *flagHTTPs == true {
		// User wants HTTPs...
		if *flagSSLCertFile == "" {
			// ...but did not provide a certificate.
			flag.Usage()
			log.Fatalf(ErrMissingSSLCert.Error())
		}
		if *flagSSLKeyFile == "" {
			// ...but did not provide the certificate key.
			flag.Usage()
			log.Fatalf(ErrMissingSSLKey.Error())
		}
	}

	// Alles gut. Initializing proxy.
	p := proxy.New()

	// Listen on which interface:port.
	if *flagListen != "" {
		p.Bind = *flagListen
	}

	// Working directory.
	if *flagWorkdir != "" {
		proxy.Workdir = *flagWorkdir
	}

	// Logs request to stdout.
	p.AddDirector(logger.Client(os.Stdout))

	// Logging different parts of the request to files.
	p.AddDirector(logger.Request)
	p.AddDirector(logger.Head)
	p.AddDirector(logger.Body)

	p.AddDirector(inject.Head)
	p.AddDirector(inject.Body)

	p.AddInterceptor(intercept.Head)
	p.AddInterceptor(intercept.Body)

	p.AddWriter(save.Head)
	p.AddWriter(save.Body)
	p.AddWriter(save.Response)

	// Logs responses to clients.
	p.AddLogger(logger.Server(os.Stdout))

	log.Printf("Hyperfox tool, by Carlos Reventlov.\n")
	log.Printf("http://www.reventlov.com\n\n")

	if *flagHTTPs == true {
		// HTTPs
		err = p.StartTLS(*flagSSLCertFile, *flagSSLKeyFile)
	} else {
		// Normal HTTP proxy.
		err = p.Start()
	}

	if err != nil {
		log.Fatalf(ErrBindFailed.Error(), err.Error())
	}
}
