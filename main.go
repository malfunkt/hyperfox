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
	"github.com/xiam/hyperfox/proxy"
	"github.com/xiam/hyperfox/tools/capture"
	"github.com/xiam/hyperfox/tools/logger"
	"log"
	"upper.io/db"
	"upper.io/db/sqlite"
)

const (
	defaultBind = `0.0.0.0:9999`
)

var (
	flagListen      = flag.String("l", defaultBind, "Listen on [address]:[port].")
	flagHTTPs       = flag.Bool("s", false, "Enable HTTPs.")
	flagSSLCertFile = flag.String("c", "", "Path to SSL certificate file.")
	flagSSLKeyFile  = flag.String("k", "", "Path to SSL key file.")
	flagWorkdir     = flag.String("o", "capture", "Working directory.")
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

	var sess db.Database

	if sess, err = db.Open(sqlite.Adapter, sqlite.ConnectionURL{Database: "capture.db"}); err != nil {
		log.Printf("SQLite: %q\n", err)
	}

	var col db.Collection

	if col, err = sess.Collection("capture"); err != nil {
		if err == db.ErrCollectionDoesNotExist {
			// Create collection.
			if sqld, ok := sess.Driver().(*sql.DB); ok {
				_, err := sqld.Exec(`CREATE TABLE "capture" (
					"id" INTEGER PRIMARY KEY,
					"remote_addr" VARCHAR(50),
					"method" VARCHAR(6),
					"status" INTEGER,
					"host" VARCHAR(255),
					"url" TEXT,
					"header" TEXT,
					"body" BLOB,
					"date" VARCHAR(20)
				)`)
				if err != nil {
					log.Printf("SQLite: %q\n", err)
					return
				}
				col, err = sess.Collection("capture")
				if err != nil {
					log.Printf("SQLite: %q\n", err)
					return
				}
			}
			log.Printf("SQLite: %q\n", err)
		}
	}

	p := proxy.NewProxy()

	p.AddLogger(logger.Stdout{})

	res := make(chan capture.Response, 256)

	p.AddBodyWriteCloser(capture.New(res))

	go func() {
		for {
			select {
			case r := <-res:
				if _, err := col.Append(r); err != nil {
					log.Printf("Sqlite: %q\n", err)
				}
			}
		}
	}()

	log.Printf("Hyperfox tool, by José Carlos Nieto.\n")
	log.Printf("http://www.reventlov.com\n\n")

	if *flagHTTPs {
		err = p.StartTLS(*flagListen)
	} else {
		err = p.Start(*flagListen)
	}

	if err != nil {
		log.Fatalf(ErrBindFailed.Error(), err.Error())
	}

}
