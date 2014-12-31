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

const version = "0.9"

const (
	defaultAddress           = `0.0.0.0`
	defaultPort              = uint(1080)
	defaultSSLPort           = uint(10443)
	defaultCaptureCollection = `capture`
	defaultDatabase          = `hyperfox-%05d.db`
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
	flagDatabase    = flag.String("d", "", "Path to database.")
	flagAddress     = flag.String("l", defaultAddress, "Bind address.")
	flagPort        = flag.Uint("p", defaultPort, "Port to bind to.")
	flagSSLPort     = flag.Uint("s", defaultSSLPort, "Port to bind to (SSL mode).")
	flagSSLCertFile = flag.String("c", "", "Path to root CA certificate.")
	flagSSLKeyFile  = flag.String("k", "", "Path to root CA key.")
)

var (
	enableDatabaseSave = false
)

var (
	sess db.Database
	col  db.Collection
)

// dbsetup sets up the database.
func dbsetup() error {
	var err error
	var databaseName string

	if *flagDatabase == "" {
		// Let's find an unused database file.
		for i := 0; ; i++ {
			databaseName = fmt.Sprintf(defaultDatabase, i)
			if _, err := os.Stat(databaseName); err != nil {
				// File does not exists (yet).
				// And that's OK.
				break
			}
		}
	} else {
		// Use the provided database name.
		databaseName = *flagDatabase
	}

	// Attempting to open database.
	if sess, err = db.Open(sqlite.Adapter, sqlite.ConnectionURL{Database: databaseName}); err != nil {
		log.Fatalf(ErrDatabaseConnection.Error(), err)
	}

	// Collection lookup.
	col, err = sess.Collection(defaultCaptureCollection)

	if err == nil {
		// Collection exists! Nothing else to do.
		log.Printf("Using database %s.", databaseName)
		return nil
	}

	log.Printf("Initializing database %s...", databaseName)

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

	return nil
}

// Parses flags and initializes Hyperfox tool.
func main() {
	var err error
	var sslEnabled bool

	// Banner.
	log.Printf("Hyperfox v%s // https://hyperfox.org\n", version)
	log.Printf("By José Carlos Nieto.\n\n")

	// Parsing command line flags.
	flag.Parse()

	// Opening database.
	if err = dbsetup(); err != nil {
		log.Fatalf("db: %q", err)
	}

	defer sess.Close()

	// Is SSL enabled?
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

	if err = startServices(); err != nil {
		log.Fatal("startServices:", err)
	}

	fmt.Println("")

	cerr := make(chan error)

	// Starting proxy servers.

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
