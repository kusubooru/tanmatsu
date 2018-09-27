package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kusubooru/tanmatsu"
	"github.com/kusubooru/tanmatsu/mysql"
)

var (
	theVersion = "devel"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var (
		user        = flag.String("user", "eribo", "username to use to connect to MySQL")
		pass        = flag.String("pass", "", "password to use to connect to MySQL")
		host        = flag.String("host", "localhost", "host for connecting to MySQL")
		port        = flag.String("port", "3306", "port for connecting to MySQL")
		dbname      = flag.String("dbname", "eribo", "database to use")
		secret      = flag.String("secret", "", "secret to decode JWT")
		issuer      = flag.String("issuer", "monban", "accepting tokens from this issuer")
		showVersion = flag.Bool("v", false, "print program version")
	)
	flag.Parse()

	version := fmt.Sprintf("%s %s (runtime: %s)", filepath.Base(os.Args[0]), theVersion, runtime.Version())
	if *showVersion {
		fmt.Println(version)
		return nil
	}

	if *secret == "" {
		return fmt.Errorf("no secret provided, use -secret")
	}

	dataSource := fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true", *user, *pass, *host, *port, *dbname)
	store, err := mysql.NewTanmatsuStore(dataSource)
	if err != nil {
		return fmt.Errorf("creating store: %v", err)
	}

	s := tanmatsu.NewServer(store, version, *issuer, []byte(*secret))
	log.Println("listening on :8080")
	return http.ListenAndServe(":8080", s)
}
