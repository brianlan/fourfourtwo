package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

var schema = `
CREATE TABLE person (
    first_name text,
    last_name text,
    email text
);
CREATE TABLE league (
	id varchar(2),
	name varchar(32)
);
`

type Person struct {
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
}

type League struct {
	Id   string `db:"id"`
	Name string `db:"name"`
}

func main() {
	db, err := sqlx.Connect("sqlite3", "foo.db")
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	db.MustExec(schema)

	newPersonArray := []Person{Person{"Rambo", "lan", "r@p.com"}, Person{"cherry", "zhou", "z@p.com"}}

	for _, p := range newPersonArray {
		_, err := db.NamedExec(`INSERT INTO person (first_name, last_name, email) VALUES (:first_name, :last_name, :email)`, p)
		if err != nil {
			log.Fatal(err)
		}
	}

	//people := []Person{}
	//db.Select(&people, "SELECT * FROM person ORDER BY first_name ASC")
	//
	//for _, p := range people {
	//	fmt.Println(p)
	//}

	leagues := []League{
		League{"23", "La Liga"},
		League{"8", "Premier League"},
		League{"21", "Serie A"},
		League{"22", "Bundesliga"},
		League{"24", "Ligue 1"},
		League{"5", "UEFA Champions League"}}
	//fmt.Println(leagues)

	for _, l := range leagues {
		_, err := db.NamedExec(`INSERT INTO league (id, name) VALUES (:id, :name)`, l)
		if err != nil {
			log.Fatal(err)
		}
	}

	var count int64
	err = db.Get(&count, "SELECT max(first_name) FROM person")
	fmt.Printf("%d\n", count)

}
