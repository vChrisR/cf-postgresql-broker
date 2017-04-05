package pgp

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/lib/pq"
)

type PGP struct {
	conn   *sql.DB
	host   string
	port   string
	prefix string
}

type Credentials struct {
	DBName   string `json:"dbname"`
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Url      string `json:"url"`
}

// defaultPort is PostgreSQL default port
const defaultPort = "5432"

// Creates new broker instance
func New(source string) (*PGP, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "postgresql" {
		return nil, errors.New("malformed url")
	}

	chunks := strings.SplitN(u.Host, ":", 2)
	if len(chunks) < 2 {
		chunks = append(chunks, defaultPort)
	}

	conn, err := connect(source)
	if err != nil {
		return nil, err
	}

	return &PGP{
		conn:   conn,
		host:   chunks[0],
		port:   chunks[1],
		prefix: "sb_",
	}, nil
}

// Creates a DB
func (b *PGP) CreateDB(ctx context.Context, d string) (string, error) {
	dbname := b.dbname(d)
	_, err := b.conn.ExecContext(ctx, "CREATE DATABASE "+de(dbname))
	return dbname, err
}

// Terminates all active connections to a DB and drops it
// If it fails to terminate them it writes errors messages to STDERR
func (b *PGP) DropDB(ctx context.Context, d string) error {
	dbname := b.dbname(d)
	if _, err := b.conn.ExecContext(ctx, "UPDATE pg_database SET datallowconn = $1 WHERE datname = $2", false, dbname); err != nil {
		return err
	}

	if _, err := b.conn.ExecContext(ctx, "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", dbname); err != nil {
		return err
	}

	// TODO: drop users
	_, err := b.conn.Exec("DROP DATABASE " + de(dbname))
	return err
}

// Creates a DB user
func (b *PGP) CreateUser(ctx context.Context, d, u string) (*Credentials, error) {
	dbname := b.dbname(d)
	if !b.databaseExists(ctx, dbname) {
		return nil, fmt.Errorf("database %q doesn't exist", dbname)
	}

	username := b.username(u)
	password, err := b.password(8)
	if err != nil {
		return nil, err
	}

	if !b.userExists(ctx, username) {
		if _, err := b.conn.ExecContext(ctx, "CREATE USER "+de(username)+" WITH PASSWORD "+se(password)); err != nil {
			return nil, err
		}
	}

	if _, err := b.conn.ExecContext(ctx, "GRANT ALL PRIVILEGES ON DATABASE "+de(dbname)+" TO "+de(username)); err != nil {
		return nil, err
	}

	return &Credentials{
		DBName:   dbname,
		Username: username,
		Password: password,
		Host:     b.host,
		Port:     b.port,
		Url:      fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", username, password, b.host, b.port, dbname),
	}, nil
}

// Drops a DB user
func (b *PGP) DropUser(ctx context.Context, d, u string) error {
	dbname := b.dbname(d)
	username := b.username(u)
	if _, err := b.conn.Exec("REVOKE ALL PRIVILEGES ON DATABASE " + de(dbname) + " FROM " + de(username)); err != nil {
		return err
	}
	_, err := b.conn.Exec("DROP USER " + de(username))
	return err
}

// Returns db name based on its instance id
func (b *PGP) dbname(d string) string {
	return b.prefix + d
}

// Returns db user name based on its instance and binding ids
func (b *PGP) username(u string) string {
	return b.prefix + u
}

// Generates random password
func (b *PGP) password(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}

// Checks whether DB exists or not
func (b *PGP) databaseExists(ctx context.Context, dbname string) bool {
	return b.exists(ctx, "pg_database", "datname", dbname)
}

// Checks whether user exists or not
func (b *PGP) userExists(ctx context.Context, username string) bool {
	return b.exists(ctx, "pg_user", "usename", username)
}

// nodoc
func (b *PGP) exists(ctx context.Context, table, column, value string) bool {
	var num string
	b.conn.QueryRowContext(ctx, "SELECT 1 FROM "+table+" WHERE "+column+" = $1", value).Scan(&num)
	return num != ""
}

// connect opens a db connection and pings it
func connect(source string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", source)
	if err != nil {
		return nil, err
	}
	return conn, conn.Ping()
}

// de double-quotes the named string safely escaping it
func de(s string) string {
	return fmt.Sprintf("%q", s)
}

// se single-quotes the named string safely escaping it
func se(s string) string {
	return "'" + strings.Replace(s, "'", "\\'", -1) + "'"
}
