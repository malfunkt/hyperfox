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
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/xiam/hyperfox/proxy"
	"github.com/xiam/hyperfox/tools/capture"
	"github.com/xiam/hyperfox/tools/logger"
	"upper.io/db"
	"upper.io/db/sqlite"
)

const (
	defaultAddress           = `0.0.0.0`
	defaultPort              = uint(1080)
	defaultSSLPort           = uint(10443)
	defaultCaptureCollection = `capture`
	defaultDatabaseFile      = `hyperfox.db`
)

const collectionCreateSQL = `CREATE TABLE "` + defaultCaptureCollection + `" (
	"id" INTEGER PRIMARY KEY,
	"origin" VARCHAR(255),
	"method" VARCHAR(10),
	"status" INTEGER,
	"content_type" VARCHAR(50),
	"content_length" INTEGER,
	"host" VARCHAR(255),
	"url" TEXT,
	"scheme" VARCHAR(10),
	"path" TEXT,
	"header" TEXT,
	"body" BLOB,
	"request_header" TEXT,
	"request_body" BLOB,
	"date_start" VARCHAR(20),
	"date_end" VARCHAR(20),
	"time_taken" INTEGER
)`

var (
	flagAddress     = flag.String("l", defaultAddress, "Bind address.")
	flagPort        = flag.Uint("p", defaultPort, "Source port to use.")
	flagSSLPort     = flag.Uint("s", defaultSSLPort, "Source port to use (SSL).")
	flagSSLCertFile = flag.String("c", "", "Path to root SSL certificate.")
	flagSSLKeyFile  = flag.String("k", "", "Path to root SSL key.")
)

var (
	enableDatabaseSave = false
)

var sess db.Database
var col db.Collection

// init sets up the database.
func init() {
	var err error

	// Attempt to open database.
	if sess, err = db.Open(sqlite.Adapter, sqlite.ConnectionURL{Database: defaultDatabaseFile}); err != nil {
		log.Fatalf(ErrDatabaseConnection.Error(), err)
	}

	// Collection lookup.
	col, err = sess.Collection(defaultCaptureCollection)

	if err == nil {
		// Collection exists! Nothing else to do.
		return
	}

	if err != db.ErrCollectionDoesNotExist {
		// This error is different to a missing collection error.
		log.Fatalf(ErrDatabaseConnection.Error(), err)
	}

	// Collection does not exists, let's create it.
	if drv, ok := sess.Driver().(*sql.DB); ok {
		// Execute CREATE TABLE.
		if _, err = drv.Exec(collectionCreateSQL); err != nil {
			log.Fatalf(ErrDatabaseConnection.Error(), err)
		}
		// Try pulling collection again.
		if col, err = sess.Collection(defaultCaptureCollection); err != nil {
			log.Fatalf(ErrDatabaseConnection.Error(), err)
		}
	}

}

// Parses flags and initializes Hyperfox tool.
func main() {
	var err error
	var sslEnabled bool

	defer sess.Close()

	flag.Parse()

	// SSL is enabled?
	if *flagSSLPort > 0 && *flagSSLCertFile != "" {
		sslEnabled = true
	}

	// User requested SSL mode.
	if sslEnabled {
		if *flagSSLCertFile == "" {
			flag.Usage()
			log.Fatal(ErrMissingSSLCert)
		}

		if *flagSSLKeyFile == "" {
			flag.Usage()
			log.Fatal(ErrMissingSSLKey)
		}

		os.Setenv(proxy.EnvSSLCert, *flagSSLCertFile)
		os.Setenv(proxy.EnvSSLKey, *flagSSLKeyFile)
	}

	// Creatig proxy.
	p := proxy.NewProxy()

	// Attaching logger.
	p.AddLogger(logger.Stdout{})

	// Attaching capture tool.
	res := make(chan capture.Response, 256)

	p.AddBodyWriteCloser(capture.New(res))

	// Saving captured data with a goroutine.
	go func() {
		for {
			select {
			case r := <-res:
				if _, err := col.Append(r); err != nil {
					log.Printf(ErrDatabaseError.Error(), err)
				}
			}
		}
	}()

	// Banner.
	log.Printf("Hyperfox // https://www.hyperfox.org\n")
	log.Printf("By José Carlos Nieto.\n\n")

	if err = startServices(); err != nil {
		log.Fatal("startServices:", err)
	}

	fmt.Println("")

	cerr := make(chan error)

	go func() {
		if err := p.Start(fmt.Sprintf("%s:%d", *flagAddress, *flagPort)); err != nil {
			cerr <- err
		}
	}()

	if sslEnabled {
		go func() {
			if err := p.StartTLS(fmt.Sprintf("%s:%d", *flagAddress, *flagSSLPort)); err != nil {
				cerr <- err
			}
		}()
	}

	err = <-cerr

	log.Fatalf(ErrBindFailed.Error(), err)
}
