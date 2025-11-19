// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package users

import (
	"database/sql"
	"fmt"
	"log/slog"
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

	CreatedAt int64
	Deleted   bool

	AllowedScopes auth.Scopes
}

func (u User) Delete() error {
	u.Deleted = true
	if err := u.h.stmtTokenDeleteAll.run(u); err != nil {
		return fmt.Errorf("unable to delete user while deleting tokens: %w", err)
	}
	return u.Update("User deleted")
}

func (u User) Update(reason string) error {
	if err := u.h.stmtUserUpdate.run(u); err != nil {
		return err
	}
	u.h.fs.Audit.AppendEvent(u.id, reason)
	return nil
}

type Storage struct {
	db *storage.DbHandle
	fs *storage.FsHandle

	hmacSecret []byte

	stmtUserCreate    stmtUserCreate
	stmtUserGetById   stmtUserGetById
	stmtUserGetByName stmtUserGetByName
	stmtUserList      stmtUserList
	stmtUserUpdate    stmtUserUpdate

	stmtTokenCreate    stmtTokenCreate
	stmtTokenDelete    stmtTokenDelete
	stmtTokenDeleteAll stmtTokenDeleteAll
	stmtTokenList      stmtTokenList
	stmtTokenLookup    stmtTokenLookup
}

func NewStorage(db *storage.DbHandle, fs *storage.FsHandle) (*Storage, error) {
	hmacSecret, err := fs.Certs.ReadFile(storage.HmacFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read HMAC secret for API tokens: %w", err)
	}
	handle := Storage{
		db:         db,
		fs:         fs,
		hmacSecret: hmacSecret,
	}

	if err := db.InitStmt(
		&handle.stmtUserCreate,
		&handle.stmtUserGetById,
		&handle.stmtUserGetByName,
		&handle.stmtUserList,
		&handle.stmtUserUpdate,
		&handle.stmtTokenCreate,
		&handle.stmtTokenDelete,
		&handle.stmtTokenDeleteAll,
		&handle.stmtTokenList,
		&handle.stmtTokenLookup,
	); err != nil {
		return nil, err
	}

	return &handle, nil
}

func (s Storage) Create(u *User) error {
	err := s.stmtUserCreate.run(u)
	if err == nil {
		u.h = s
		s.fs.Audit.AppendEvent(u.id, "User created")
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

func (s Storage) List() ([]User, error) {
	users, err := s.stmtUserList.run()
	if err == nil {
		for i := range users {
			users[i].h = s
		}
	}
	return users, err
}

type stmtUserCreate storage.DbStmt

func (s *stmtUserCreate) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("userCreate", `
		INSERT INTO users (username, password, email, created_at, deleted, allowed_scopes)
		VALUES (?, ?, ?, ?, ?, ?)`,
	)
	return
}

func (s *stmtUserCreate) run(u *User) error {
	if u.CreatedAt == 0 {
		u.CreatedAt = time.Now().Unix()
	}
	result, err := s.Stmt.Exec(
		u.Username,
		u.Password,
		u.Email,
		u.CreatedAt,
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

type stmtUserGetById storage.DbStmt

func (s *stmtUserGetById) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("userGetId", `
		SELECT id, username, password, email, created_at, allowed_scopes
		FROM users
		WHERE id = ? and deleted = false`,
	)
	return
}

func (s *stmtUserGetById) run(id int64) (*User, error) {
	u := User{}
	err := s.Stmt.QueryRow(id).Scan(
		&u.id,
		&u.Username,
		&u.Password,
		&u.Email,
		&u.CreatedAt,
		&u.AllowedScopes,
	)
	return &u, err
}

type stmtUserGetByName storage.DbStmt

func (s *stmtUserGetByName) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("userGet", `
		SELECT id, username, password, email, created_at, allowed_scopes
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
		&u.CreatedAt,
		&u.AllowedScopes,
	)
	return &u, err
}

type stmtUserList storage.DbStmt

func (s *stmtUserList) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("userList", `
		SELECT id, username, password, email, created_at, deleted, allowed_scopes
		FROM users
		WHERE deleted = false`,
	)
	return
}

func (s *stmtUserList) run() ([]User, error) {
	var users []User
	rows, err := s.Stmt.Query()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("stmtUserList: failed to close rows", "error", err)
		}
	}()

	for rows.Next() {
		var u User
		err := rows.Scan(
			&u.id,
			&u.Username,
			&u.Password,
			&u.Email,
			&u.CreatedAt,
			&u.Deleted,
			&u.AllowedScopes,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

type stmtUserUpdate storage.DbStmt

func (s *stmtUserUpdate) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("userUpdate", `
		UPDATE users
		SET username = ?, password = ?, email = ?, allowed_scopes = ?, deleted = ?
		WHERE id = ?`,
	)
	return
}

func (s *stmtUserUpdate) run(u User) error {
	_, err := s.Stmt.Exec(u.Username, u.Password, u.Email, u.AllowedScopes, u.Deleted, u.id)
	return err
}
