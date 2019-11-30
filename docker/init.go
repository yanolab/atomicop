package main

import (
	"database/sql"
	"io/ioutil"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "root:pass@tcp(127.0.0.1:3306)/atomicop")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	b, err := ioutil.ReadFile("docker/schema.sql")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(string(b)); err != nil {
		log.Fatal(err)
	}
}
