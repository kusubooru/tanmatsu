package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/kusubooru/tanmatsu"
	"github.com/kusubooru/tanmatsu/mysql"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var (
		user   = flag.String("user", "eribo", "username to use to connect to MySQL")
		pass   = flag.String("pass", "", "password to use to connect to MySQL")
		host   = flag.String("host", "localhost", "host for connecting to MySQL")
		port   = flag.String("port", "3306", "port for connecting to MySQL")
		dbname = flag.String("dbname", "eribo", "database to use")
	)
	flag.Parse()

	dataSource := fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true", *user, *pass, *host, *port, *dbname)
	store, err := mysql.NewTanmatsuStore(dataSource)
	if err != nil {
		return fmt.Errorf("creating store: %v", err)
	}

	s := tanmatsu.NewServer(store)
	log.Println("listening on :8080")
	return http.ListenAndServe(":8080", s)
}
