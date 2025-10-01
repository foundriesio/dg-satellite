// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package storage

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type DbHandle struct {
	db *sql.DB
}

func NewDb(dbfile string) (*DbHandle, error) {
	var newDb bool
	if _, err := os.Stat(dbfile); err != nil {
		newDb = os.IsNotExist(err)
	}
	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		return nil, err
	}
	if newDb {
		if err := createTables(db); err != nil {
			return nil, err
		}
	}
	return &DbHandle{db: db}, nil
}

func (d DbHandle) Close() error {
	return d.db.Close()
}

func (d DbHandle) Prepare(name, query string) (stmt *sql.Stmt, err error) {
	if stmt, err = d.db.Prepare(query); err != nil {
		err = fmt.Errorf("unable to prepare '%s' statement: %w", name, err)
	}
	return
}

func (d DbHandle) InitStmt(stmt ...DbStmtInit) (err error) {
	for _, s := range stmt {
		if err = s.Init(d); err != nil {
			break
		}
	}
	return
}

func createTables(db *sql.DB) error {
	sqlStmt := `
		CREATE TABLE devices (
			uuid VARCHAR(48) NOT NULL PRIMARY KEY,
			pubkey TEXT,
			deleted BOOL,
			is_prod BOOL,
			created_at INT DEFAULT 0,
			last_seen INT DEFAULT 0,
			tag VARCHAR(80) DEFAULT "",
			group_name VARCHAR(80) DEFAULT "",
			update_name VARCHAR(80) DEFAULT "",
			target_name VARCHAR(80) DEFAULT "",
			ostree_hash VARCHAR(80) DEFAULT "",
			apps VARCHAR(2048) DEFAULT ""
		) WITHOUT ROWID;
	`
	if _, err := db.Exec(sqlStmt); err != nil {
		return fmt.Errorf("unable to create devices db: %w", err)
	}
	return nil
}

type DbStmt struct {
	Stmt *sql.Stmt
}

type DbStmtInit interface {
	Init(db DbHandle) error
}
