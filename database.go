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
	"uuid" VARCHAR(36) NOT NULL,
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
	"keywords" BLOB,
	"request_header" TEXT,
	"request_body" BLOB,
	"date_start" DATETIME,
	"date_end" DATETIME,
	"time_taken" INTEGER
)`

func initDB() (db.Database, error) {

	databaseName := *flagDatabase
	if databaseName == "" {
		// Let's find an unused database file.
		for i := 0; ; i++ {
			databaseName = fmt.Sprintf(defaultDatabase, i)
			if _, err := os.Stat(databaseName); err != nil {
				// File does not exists (yet).
				// And that's OK.
				break
			}
		}
	}

	// Attempting to open database.
	sess, err := sqlite.Open(sqlite.ConnectionURL{Database: databaseName})
	if err != nil {
		return nil, err
	}

	log.Printf("Using SQLite database: %s", databaseName)

	// Collection lookup.
	col := sess.Collection(defaultCaptureCollection)
	if col.Exists() {
		return sess, nil
	}

	// Collection does not exists, let's create it.
	// Execute CREATE TABLE.
	if _, err = sess.Exec(collectionCreateSQL); err != nil {
		return nil, err
	}

	sess.ClearCache()

	return sess, nil
}
