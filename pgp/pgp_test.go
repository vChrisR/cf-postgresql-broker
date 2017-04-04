package pgp

import (
	"os"
	"testing"
)

const testDB = "test_foo"
const testUser = "test_bar"

func TestNew(t *testing.T) {
	if _, err := newPGP(t); err != nil {
		t.Fatal("cannot connect to DB", err)
	}

	if _, err := New("foo"); err == nil {
		t.Fatal("has not returned error")
	}
}

func TestCreateAndDropDB(t *testing.T) {
	pgp, _ := newPGP(t)
	dbname := pgp.dbname(testDB)

	defer pgp.exec(`DROP DATABASE ?`, dbname)

	dbname, err := pgp.CreateDB(testDB)

	if err != nil {
		t.Fatal(err)
	}

	if dbname == "" {
		t.Fatal("dbname is empty")
	}

	if !pgp.databaseExists(dbname) {
		t.Fatal("database doesn't exist")
	}

	if err := pgp.DropDB(testDB); err != nil {
		t.Fatal(err)
	}

	if pgp.databaseExists(dbname) {
		t.Fatal("database still exists")
	}
}

func TestCreateAndDropUser(t *testing.T) {
	pgp, _ := newPGP(t)
	dbname := pgp.dbname(testDB)
	username := pgp.username(testUser)

	defer func() {
		pgp.exec(`REVOKE ALL PRIVILEGES ON DATABASE ? FROM ?`, dbname, username)
		pgp.exec(`DROP USER ?`, username)
		pgp.exec(`DROP DATABASE ?`, dbname)
	}()

	if _, err := pgp.CreateDB(testDB); err != nil {
		t.Fatal(err)
	}

	creds, err := pgp.CreateUser(testDB, testUser)

	if err != nil {
		t.Fatal(err)
	}

	if creds == nil {
		t.Fatal("credentials are blank")
	}

	if !pgp.userExists(username) {
		t.Fatal("user hasn't been created")
	}

	if err := pgp.DropUser(testDB, testUser); err != nil {
		t.Fatal(err)
	}

	if pgp.userExists(username) {
		t.Fatal("user still exists")
	}
}

func newPGP(t *testing.T) (*PGPuppeteer, error) {
	source := os.Getenv("PG_SOURCE")

	if source == "" {
		t.Fatal("Environment variable PG_SOURCE is not provided or empty")
	}

	return New(os.Getenv("PG_SOURCE"))
}
