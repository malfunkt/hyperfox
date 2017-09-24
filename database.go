package main

import (
	"fmt"
	"log"
	"os"

	"upper.io/db.v3"
	"upper.io/db.v3/sqlite"
)

const (
	defaultCaptureCollection = `capture`
	defaultDatabase          = `hyperfox-%05d.db`
)

const collectionCreateSQL = `CREATE TABLE "` + defaultCaptureCollection + `" (
	"id" INTEGER PRIMARY KEY,
	"origin" VARCHAR(255),
	"method" VARCHAR(10),
	"status" INTEGER,
	"content_type" VARCHAR(255),
	"content_length" INTEGER,
	"host" VARCHAR(255),
	"url" TEXT,
	"scheme" VARCHAR(10),
	"path" TEXT,
	"header" TEXT,
	"body" BLOB,
	"request_header" TEXT,
	"request_body" BLOB,
	"date_start" DATETIME,
	"date_end" DATETIME,
	"time_taken" INTEGER
)`

func dbInit() (db.Database, error) {
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
	sess, err := sqlite.Open(sqlite.ConnectionURL{Database: databaseName})
	if err != nil {
		return nil, err
	}

	// Collection lookup.
	col := sess.Collection(defaultCaptureCollection)
	if col.Exists() {
		return sess, nil
	}

	log.Printf("Initializing database %s...", databaseName)
	// Collection does not exists, let's create it.
	// Execute CREATE TABLE.
	if _, err = sess.Exec(collectionCreateSQL); err != nil {
		return nil, err
	}

	return sess, nil
}
