// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package users

import (
	"database/sql"
	"time"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/storage"
)

type User struct {
	h  Storage
	id int64

	Username string
	Password string
	Email    string

	Created storage.Timestamp
	Deleted bool

	AllowedScopes auth.Scopes
}

type Storage struct {
	db *storage.DbHandle
	fs *storage.FsHandle

	stmtUserCreate    stmtUserCreate
	stmtUserGetByName stmtUserGetByName
}

func NewStorage(db *storage.DbHandle, fs *storage.FsHandle) (*Storage, error) {
	handle := Storage{
		db: db,
		fs: fs,
	}

	if err := db.InitStmt(
		&handle.stmtUserCreate,
		&handle.stmtUserGetByName,
	); err != nil {
		return nil, err
	}

	return &handle, nil
}

func (s Storage) Create(u *User) error {
	err := s.stmtUserCreate.run(u)
	if err == nil {
		u.h = s
	}
	return err
}

func (s Storage) Get(username string) (*User, error) {
	u, err := s.stmtUserGetByName.run(username)
	switch err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		u.h = s
	}
	return u, err
}

type stmtUserCreate storage.DbStmt

func (s *stmtUserCreate) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("userCreate", `
		INSERT INTO users (username, password, email, created, deleted, allowed_scopes)
		VALUES (?, ?, ?, ?, ?, ?)`,
	)
	return
}

func (s *stmtUserCreate) run(u *User) error {
	if u.Created == 0 {
		u.Created = storage.Timestamp(time.Now().Unix())
	}
	result, err := s.Stmt.Exec(
		u.Username,
		u.Password,
		u.Email,
		u.Created,
		u.Deleted,
		u.AllowedScopes,
	)
	if err != nil {
		return err
	} else if id, err := result.LastInsertId(); err != nil {
		return err
	} else {
		u.id = id
	}
	return nil
}

type stmtUserGetByName storage.DbStmt

func (s *stmtUserGetByName) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("userGet", `
		SELECT id, username, password, email, created, allowed_scopes
		FROM users
		WHERE username = ? AND deleted = false`,
	)
	return
}

func (s *stmtUserGetByName) run(username string) (*User, error) {
	u := User{}
	err := s.Stmt.QueryRow(username).Scan(
		&u.id,
		&u.Username,
		&u.Password,
		&u.Email,
		&u.Created,
		&u.AllowedScopes,
	)
	return &u, err
}
