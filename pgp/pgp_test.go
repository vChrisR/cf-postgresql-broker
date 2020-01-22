package pgp

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
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
	pgp, err := newPGP(t)
	if err != nil {
		t.Fatal(err)
	}

	dbname := pgp.dbname(testDB)

	defer pgp.conn.Exec(`DROP DATABASE $1`, dbname)

	dbname, err = pgp.CreateDB(context.Background(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	if dbname == "" {
		t.Fatal("dbname is empty")
	}

	if !pgp.DatabaseExists(context.Background(), dbname) {
		t.Fatal("database doesn't exist")
	}

	if err := pgp.DropDB(context.Background(), testDB); err != nil {
		t.Fatal(err)
	}

	if pgp.DatabaseExists(context.Background(), dbname) {
		t.Fatal("database still exists")
	}
}

func TestCreateAndDropUser(t *testing.T) {
	pgp, err := newPGP(t)
	if err != nil {
		t.Fatal(err)
	}

	username := pgp.username(testUser)
	if _, err := pgp.CreateDB(context.Background(), testDB); err != nil {
		t.Fatal(err)
	}
	defer pgp.DropDB(context.Background(), testDB)

	creds, err := pgp.CreateUser(context.Background(), testDB, testUser)
	if err != nil {
		t.Fatal(err)
	}

	if creds == nil {
		t.Fatal("credentials are blank")
	}

	if !pgp.userExists(context.Background(), username) {
		t.Fatal("user hasn't been created")
	}

	// Create a table owned by that user, to ensure we can drop users that own
	// tables.  Do this in a function to ensure we close the connection early.
	func() {
		conn, err := sql.Open("postgres", creds.Url)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		_, err = conn.Exec("CREATE TABLE IF NOT EXISTS " + testUser + "_table()")
		if err != nil {
			t.Fatalf("Failed to create table: %s", err)
		}
	}()
	defer pgp.conn.Exec("DROP TABLE IF EXISTS foo")

	if err := pgp.DropUser(context.Background(), testDB, testUser); err != nil {
		t.Fatal(err)
	}

	if pgp.userExists(context.Background(), username) {
		t.Fatal("user still exists")
	}
}

func newPGP(t *testing.T) (*PGP, error) {
	source := os.Getenv("PG_SOURCE")
	if source == "" {
		t.Fatal("$PG_SOURCE is required")
	}

	return New(os.Getenv("PG_SOURCE"))
}
