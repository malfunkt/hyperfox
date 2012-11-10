/*
	Hyperfox

	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package main

import (
	"flag"
	"fmt"
	"github.com/xiam/hyperfox/proxy"
	"github.com/xiam/hyperfox/tools/inject"
	"github.com/xiam/hyperfox/tools/intercept"
	"github.com/xiam/hyperfox/tools/logger"
	"github.com/xiam/hyperfox/tools/save"
	"log"
	"os"
)

var flagListen = flag.String("l", "0.0.0.0:9999", "Listen on address:port.")
var flagHttps = flag.Bool("s", false, "Serve (bogus) HTTPs.")
var flagSslCertFile = flag.String("c", "ssl/cert.pem", "Bogus HTTP cert file.")
var flagSslKeyFile = flag.String("k", "ssl/key.pem", "Bogus HTTP key file.")
var flagWorkdir = flag.String("o", "archive", "Working directory.")

/*
	Parses flags and pass them to the hyperfox's proxy package.
*/
func main() {
	flag.Parse()

	if *flagHttps == true {
		if *flagSslCertFile == "" {
			fmt.Errorf("Missing SSL cert file.")
			return
		}
		if *flagSslKeyFile == "" {
			fmt.Errorf("Missing SSL key file.")
			return
		}
	}

	p := proxy.New()

	if *flagListen != "" {
		p.Bind = *flagListen
	}

	if *flagWorkdir != "" {
		proxy.Workdir = *flagWorkdir
	}

	p.AddDirector(logger.Client(os.Stdout))

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

	p.AddLogger(logger.Server(os.Stdout))

	var err error

	if *flagHttps {
		err = p.StartTLS(*flagSslCertFile, *flagSslKeyFile)
	} else {
		err = p.Start()
	}

	if err != nil {
		log.Printf(fmt.Sprintf("Failed to bind: %s.\n", err.Error()))
	}
}
