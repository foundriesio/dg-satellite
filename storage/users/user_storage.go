// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package users

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/storage"
)

type Token struct {
	PublicID    uint32
	Created     storage.Timestamp
	Expires     storage.Timestamp
	Description string
	Scopes      auth.Scopes
	Value       string
}

type session struct {
	UserID   int64
	RemoteIP string
	Expires  storage.Timestamp
	Scopes   auth.Scopes
}

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

func (u User) GenerateToken(description string, expires int64, scopes auth.Scopes) (*Token, error) {
	if scopes&u.AllowedScopes != scopes {
		return nil, fmt.Errorf("requested scopes %s exceed allowed scopes %s", scopes.String(), u.AllowedScopes.String())
	}

	value := "pat_" + rand.Text()

	hasher := hmac.New(sha256.New, u.h.hmacSecret)
	if _, err := hasher.Write([]byte(value)); err != nil {
		return nil, fmt.Errorf("unable to hash token value: %w", err)
	}
	hashed := fmt.Sprintf("%x", hasher.Sum(nil))

	t := Token{
		Created:     storage.Timestamp(time.Now().Unix()),
		Expires:     storage.Timestamp(expires),
		Description: description,
		Scopes:      scopes,
		Value:       hashed,
	}

	if err := u.h.stmtTokenCreate.run(u, &t); err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("Token created (id=%d, expires=%d, scopes=%s)", t.PublicID, expires, scopes)
	u.h.fs.Audit.AppendEvent(u.id, msg)
	t.Value = value
	return &t, nil
}

func (u User) DeleteToken(id uint32) error {
	if err := u.h.stmtTokenDelete.run(u, id); err != nil {
		return err
	}
	msg := fmt.Sprintf("Token deleted id=%d", id)
	u.h.fs.Audit.AppendEvent(u.id, msg)
	return nil
}

func (u User) ListTokens() ([]Token, error) {
	return u.h.stmtTokenList.run(u)
}

func (u User) CreateSession(remoteIP string, expires int64, scopes auth.Scopes) (string, error) {
	if scopes&u.AllowedScopes != scopes {
		return "", fmt.Errorf("requested scopes %s exceed allowed scopes %s", scopes.String(), u.AllowedScopes.String())
	}
	idStr := rand.Text()
	if err := u.h.stmtSessionCreate.run(u, idStr, remoteIP, time.Now().Unix(), expires, scopes); err != nil {
		return "", fmt.Errorf("unable to create session: %w", err)
	}

	msg := fmt.Sprintf("Session created (ip=%s, expires=%d, scopes=%s)", remoteIP, expires, scopes)
	u.h.fs.Audit.AppendEvent(u.id, msg)
	return idStr, nil
}

func (u User) DeleteSession(id string) error {
	if err := u.h.stmtSessionDelete.run(id); err != nil {
		return fmt.Errorf("unable to delete session: %w", err)
	}
	msg := fmt.Sprintf("Session deleted id=%s", id)
	u.h.fs.Audit.AppendEvent(u.id, msg)
	return nil
}

type Storage struct {
	db *storage.DbHandle
	fs *storage.FsHandle

	hmacSecret []byte

	stmtUserCreate           stmtUserCreate
	stmtUserGetById          stmtUserGetById
	stmtUserGetByName        stmtUserGetByName
	stmtUserList             stmtUserList
	stmtUserUpdate           stmtUserUpdate
	stmtSessionCreate        stmtSessionCreate
	stmtSessionDelete        stmtSessionDelete
	stmtSessionDeleteExpired stmtSessionDeleteExpired
	stmtSessionGet           stmtSessionGet
	stmtTokenCreate          stmtTokenCreate
	stmtTokenDelete          stmtTokenDelete
	stmtTokenDeleteAll       stmtTokenDeleteAll
	stmtTokenDeleteExpired   stmtTokenDeleteExpired
	stmtTokenList            stmtTokenList
	stmtTokenLookup          stmtTokenLookup

	done chan struct{}
}

func NewStorage(db *storage.DbHandle, fs *storage.FsHandle) (*Storage, error) {
	hmacSecret, err := fs.Certs.ReadFile("hmac.secret")
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
		&handle.stmtSessionCreate,
		&handle.stmtSessionDelete,
		&handle.stmtSessionDeleteExpired,
		&handle.stmtSessionGet,
		&handle.stmtTokenCreate,
		&handle.stmtTokenDelete,
		&handle.stmtTokenDeleteAll,
		&handle.stmtTokenDeleteExpired,
		&handle.stmtTokenList,
		&handle.stmtTokenLookup,
	); err != nil {
		return nil, err
	}

	return &handle, nil
}

func (s Storage) StartGc() {
	s.done = make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runGc()
			case <-s.done:
				slog.Info("Stopping user GC")
				return
			}
		}
	}()
}

func (s *Storage) StopGc() {
	close(s.done)
}

func (s Storage) runGc() {
	now := time.Now().Unix()
	slog.Info("Running user token GC")
	if err := s.stmtTokenDeleteExpired.run(now); err != nil {
		slog.Error("Unable to run user token GC", "error", err)
	}

	slog.Info("Running user session GC")
	if err := s.stmtSessionDeleteExpired.run(now); err != nil {
		slog.Error("Unable to run user session GC", "error", err)
	}
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

func (s Storage) GetByToken(token string) (*User, error) {
	hasher := hmac.New(sha256.New, s.hmacSecret)
	if _, err := hasher.Write([]byte(token)); err != nil {
		return nil, fmt.Errorf("unable to hash token value: %w", err)
	}
	hashed := fmt.Sprintf("%x", hasher.Sum(nil))
	t, userID, err := s.stmtTokenLookup.run(hashed)
	if err != nil {
		return nil, err
	} else if t == nil {
		return nil, nil
	}

	if t.Expires.ToTime().Before(time.Now()) {
		return nil, nil
	}
	u, err := s.stmtUserGetById.run(userID)
	if u != nil {
		u.h = s
		u.AllowedScopes = t.Scopes & u.AllowedScopes
	}
	return u, err
}

func (s Storage) GetBySession(id string) (*User, error) {
	sess, err := s.stmtSessionGet.run(id)
	if err != nil {
		return nil, err
	} else if sess == nil {
		return nil, nil
	}
	if sess.Expires.ToTime().Before(time.Now()) {
		return nil, nil
	}
	u, err := s.stmtUserGetById.run(sess.UserID)
	if u != nil {
		u.h = s
		u.AllowedScopes = sess.Scopes & u.AllowedScopes
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

type stmtUserGetById storage.DbStmt

func (s *stmtUserGetById) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("userGetId", `
		SELECT id, username, password, email, created, allowed_scopes
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
		&u.Created,
		&u.AllowedScopes,
	)
	return &u, err
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

type stmtUserList storage.DbStmt

func (s *stmtUserList) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("userList", `
		SELECT id, username, password, email, created, deleted, allowed_scopes
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
			&u.Created,
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

type stmtSessionCreate storage.DbStmt

func (s *stmtSessionCreate) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("sessionCreate", `
		INSERT INTO session (id, user_id, remote_ip, created, expires, scopes)
		VALUES (?, ?, ?, ?, ?, ?)`,
	)
	return
}

func (s *stmtSessionCreate) run(u User, id, remoteIP string, created, expires int64, scopes auth.Scopes) error {
	_, err := s.Stmt.Exec(
		id,
		u.id,
		remoteIP,
		created,
		expires,
		scopes,
	)
	return err
}

type stmtSessionDelete storage.DbStmt

func (s *stmtSessionDelete) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("sessionDelete", `
		DELETE FROM session
		WHERE id = ?`,
	)
	return
}

func (s *stmtSessionDelete) run(id string) error {
	_, err := s.Stmt.Exec(id)
	return err
}

type stmtSessionDeleteExpired storage.DbStmt

func (s *stmtSessionDeleteExpired) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("sessionDeleteExpired", `
		DELETE FROM session
		WHERE expires < ?`,
	)
	return
}

func (s *stmtSessionDeleteExpired) run(before int64) error {
	_, err := s.Stmt.Exec(before)
	return err
}

type stmtSessionGet storage.DbStmt

func (s *stmtSessionGet) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("sessionGet", `
		SELECT user_id, expires, scopes
		FROM session
		WHERE id = ?`,
	)
	return
}

func (s *stmtSessionGet) run(id string) (*session, error) {
	var sess session
	err := s.Stmt.QueryRow(id).Scan(
		&sess.UserID,
		&sess.Expires,
		&sess.Scopes,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &sess, nil
}

type stmtTokenCreate storage.DbStmt

func (s *stmtTokenCreate) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("tokenCreate", `
		INSERT INTO tokens (user_id, public_id, created, expires, description, scopes, value)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	return
}

func generateTimestampTokenID() (uint32, error) {
	// Use lower 22 bits for timestamp
	// plus 10 bits for random data
	now := uint32(time.Now().Unix()) & 0x3FFFFF // (1<<22 -1)

	var randomBuf [2]byte
	if _, err := rand.Read(randomBuf[:]); err != nil {
		return 0, err
	}
	random := binary.BigEndian.Uint16(randomBuf[:]) & 0x3FF // 10 bits

	return (now << 10) | uint32(random), nil
}

func (s *stmtTokenCreate) run(u User, t *Token) error {
	var lastErr error
	for range 10 {
		var err error
		t.PublicID, err = generateTimestampTokenID()
		if err != nil {
			return fmt.Errorf("unable to generate token ID: %w", err)
		}
		_, err = s.Stmt.Exec(
			u.id,
			t.PublicID,
			t.Created,
			t.Expires,
			t.Description,
			t.Scopes,
			t.Value,
		)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

type stmtTokenDelete storage.DbStmt

func (s *stmtTokenDelete) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("tokenDelete", `
		DELETE FROM tokens
		WHERE user_id = ? and public_id = ?`,
	)
	return
}

func (s *stmtTokenDelete) run(u User, id uint32) error {
	_, err := s.Stmt.Exec(u.id, id)
	return err
}

type stmtTokenDeleteAll storage.DbStmt

func (s *stmtTokenDeleteAll) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("tokenDeleteAll", `
		DELETE FROM tokens
		WHERE user_id = ?`,
	)
	return
}

func (s *stmtTokenDeleteAll) run(u User) error {
	_, err := s.Stmt.Exec(u.id)
	return err
}

type stmtTokenDeleteExpired storage.DbStmt

func (s *stmtTokenDeleteExpired) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("tokenDeleteExpired", `
		DELETE FROM tokens
		WHERE expires < ?`,
	)
	return
}

func (s *stmtTokenDeleteExpired) run(before int64) error {
	_, err := s.Stmt.Exec(before)
	return err
}

type stmtTokenList storage.DbStmt

func (s *stmtTokenList) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("tokenList", `
		SELECT public_id, created, expires, description, scopes
		FROM tokens
		WHERE user_id = ?
		ORDER BY created ASC`,
	)
	return
}

func (s *stmtTokenList) run(u User) ([]Token, error) {
	var tokens []Token
	rows, err := s.Stmt.Query(u.id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("stmtTokenList: failed to close rows", "error", err)
		}
	}()

	for rows.Next() {
		var t Token
		err := rows.Scan(
			&t.PublicID,
			&t.Created,
			&t.Expires,
			&t.Description,
			&t.Scopes,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

type stmtTokenLookup storage.DbStmt

func (s *stmtTokenLookup) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("tokenLookup", `
		SELECT user_id, public_id, created, expires, scopes
		FROM tokens
		WHERE value = ?`,
	)
	return
}

func (s *stmtTokenLookup) run(value string) (*Token, int64, error) {
	var t Token
	var userID int64
	err := s.Stmt.QueryRow(value).Scan(
		&userID,
		&t.PublicID,
		&t.Created,
		&t.Expires,
		&t.Scopes,
	)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, err
	}
	return &t, userID, nil
}
