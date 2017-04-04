package pgp

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

type (
	// Represents puppeteer itself
	PGPuppeteer struct {
		conn   *sql.DB
		host   string
		port   string
		prefix string
	}

	// Represents connection credentials
	Credentials struct {
		DBName   string `json:"dbname"`
		Username string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		Url      string `json:"url"`
	}
)

// PostgreSQL default port
const defaultPort = "5432"

// Returned when source is not an url
var ErrInvalidSource = errors.New("source must be an url, e.g postgresql://user:pass@localhost:5432/postgres")

// Creates new broker instance
func New(source string) (*PGPuppeteer, error) {
	if !strings.HasPrefix(source, "postgresql://") {
		return nil, ErrInvalidSource
	}

	host, port, err := parseHostAndPort(source)
	if err != nil {
		return nil, err
	}

	conn, err := connect(source)
	if err != nil {
		return nil, err
	}

	return &PGPuppeteer{
		conn:   conn,
		host:   host,
		port:   port,
		prefix: "sb_",
	}, nil
}

// Creates a DB
func (b *PGPuppeteer) CreateDB(d string) (string, error) {
	dbname := b.dbname(d)
	if err := b.exec(`CREATE DATABASE ?`, dbname); err != nil {
		return "", err
	}

	return dbname, nil
}

// Terminates all active connections to a DB and drops it
// If it fails to terminate them it writes errors messages to STDERR
func (b *PGPuppeteer) DropDB(d string) error {
	dbname := b.dbname(d)
	if _, err := b.conn.Exec("UPDATE pg_database SET datallowconn = 'false' WHERE datname = $1", dbname); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	if _, err := b.conn.Exec("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", dbname); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	// TODO: drop users
	if err := b.exec(`DROP DATABASE ?`, dbname); err != nil {
		return err
	}

	return nil
}

// Creates a DB user
func (b *PGPuppeteer) CreateUser(d, u string) (*Credentials, error) {
	dbname := b.dbname(d)
	if !b.databaseExists(dbname) {
		return nil, fmt.Errorf("database %q doesn't exist", dbname)
	}

	username := b.username(u)
	password := b.password(8)

	if !b.userExists(username) {
		if err := b.exec(fmt.Sprintf(`CREATE USER ? WITH PASSWORD '%s'`, password), username); err != nil {
			return nil, err
		}
	}

	if err := b.exec(`GRANT ALL PRIVILEGES ON DATABASE ? TO ?`, dbname, username); err != nil {
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
func (b *PGPuppeteer) DropUser(d, u string) error {
	dbname := b.dbname(d)
	username := b.username(u)
	if err := b.exec(`REVOKE ALL PRIVILEGES ON DATABASE ? FROM ?`, dbname, username); err != nil {
		return err
	}

	if err := b.exec(`DROP USER ?`, username); err != nil {
		return err
	}

	return nil
}

// Shortcut for queries
func (b *PGPuppeteer) exec(query string, args ...interface{}) error {
	query = strings.Replace(query, "?", "%s", -1)

	for i, v := range args {
		args[i] = `"` + strings.Replace(v.(string), `"`, `\"`, -1) + `"`
	}

	if _, err := b.conn.Exec(fmt.Sprintf(query, args...)); err != nil {
		return err
	}

	return nil
}

// Returns db name based on its instance id
func (b *PGPuppeteer) dbname(d string) string {
	return b.prefix + d
}

// Returns db user name based on its instance and binding ids
func (b *PGPuppeteer) username(u string) string {
	return b.prefix + u
}

// Generates random password
func (b *PGPuppeteer) password(size int) string {
	buf := make([]byte, size)

	if _, err := rand.Read(buf); err != nil {
		fmt.Println(err)
	}

	return base64.URLEncoding.EncodeToString(buf)
}

// Checks whether DB exists or not
func (b *PGPuppeteer) databaseExists(dbname string) bool {
	return b.exists("pg_database", "datname", dbname)
}

// Checks whether user exists or not
func (b *PGPuppeteer) userExists(username string) bool {
	return b.exists("pg_user", "usename", username)
}

// nodoc
func (b *PGPuppeteer) exists(table, column, value string) bool {
	var num string
	b.conn.QueryRow("SELECT 1 FROM "+table+" WHERE "+column+" = $1", value).Scan(&num)
	return num != ""
}

// Connects to DB and checks the connection
func connect(source string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", source)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return conn, nil
}

// Parses host and port out of connection url
func parseHostAndPort(source string) (string, string, error) {
	u, err := url.Parse(source)
	if err != nil {
		return "", "", err
	}

	h, p, err := net.SplitHostPort(u.Host)
	if err != nil {
		return u.Host, defaultPort, nil
	}

	return h, p, nil
}
