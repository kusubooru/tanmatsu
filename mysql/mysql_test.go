package mysql

import (
	"context"
	"flag"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kusubooru/tanmatsu"
)

var (
	user   = flag.String("user", "eribo", "username to use to run tests on MySQL")
	pass   = flag.String("pass", "", "password to use to run tests on MySQL")
	host   = flag.String("host", "localhost", "host for connecting to MySQL and run the tests")
	port   = flag.String("port", "3306", "port for connecting to MySQL and run the tests")
	dbname = flag.String("dbname", "eribo_test", "test database to use to run the tests")
)

func init() {
	flag.Parse()
}

func setup(t *testing.T) *TanmatsuStore {
	if *pass == "" {
		t.Logf("No password provided for user %q to connect to MySQL and run the tests.", *user)
		t.Logf("These tests need a MySQL account %q that has access to test database %q.", *user, *dbname)
		t.Skipf("Use: go test -pass '<db password>'")
	}
	datasource := fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true", *user, *pass, *host, *port, *dbname)
	s, err := NewTanmatsuStore(datasource)
	if err != nil {
		t.Fatalf("NewTanmatsuStore failed for datasource %q: %v", datasource, err)
	}
	if err := s.createSchema(); err != nil {
		t.Fatalf("Schema creation failed: %v", err)
	}
	return s
}

func teardown(t *testing.T, s *TanmatsuStore) {
	if err := s.dropSchema(); err != nil {
		t.Error(err)
	}
}

func TestGetImages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	store := setup(t)
	defer teardown(t, store)
	err := store.createImage(context.Background(), &tanmatsu.Image{URL: "http://google.com", Player: "john"})
	if err != nil {
		t.Fatal("create image failed:", err)
	}

	images, err := store.GetImages(context.Background(), 10, 0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(images), 1; got != want {
		t.Fatalf("len(images) = %d, want %d", got, want)
	}
	img := images[0]
	if got, want := img.URL, "http://google.com"; got != want {
		t.Errorf("img url = %s, want %s", got, want)
	}
	if got, want := img.Player, "john"; got != want {
		t.Errorf("img player = %s, want %s", got, want)
	}
}
