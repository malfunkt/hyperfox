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
var flagSslCertFile = flag.String("c", "", "Path to SSL certificate file.")
var flagSslKeyFile = flag.String("k", "", "Path to SSL key file.")
var flagWorkdir = flag.String("o", "archive", "Working directory.")

/*
	Parses flags and pass them to the hyperfox's proxy package.
*/
func main() {
	flag.Parse()

	if *flagHttps == true {
		if *flagSslCertFile == "" {
			flag.Usage()
			fmt.Printf("Missing SSL cert file.\n")
			return
		}
		if *flagSslKeyFile == "" {
			flag.Usage()
			fmt.Printf("Missing SSL key file.\n")
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
