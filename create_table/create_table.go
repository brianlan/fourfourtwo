package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

var schema = `
CREATE TABLE player (
    id varchar(8),
    name varchar(128)
);

CREATE TABLE match (
	id varchar(8),
	season varchar(4),
	match_date varchar(10),
	match_time varchar(10),
	league_id varchar(2),
	home_team_name varchar(64),
	away_team_name varchar(64),
	home_score varchar(2),
	away_score varchar(2),
	url varchar(512),
	is_crawled varchar(1)
);

CREATE TABLE league (
	id varchar(2),
	name varchar(32)
);

CREATE TABLE player_stats (
	id integer primary key,
	match_id varchar(8),
	team_name varchar(64),
	player_id varchar(8),
	player_name varchar(128),
	is_substitute varchar(1),
	url varchar(512)
);

CREATE TABLE player_event (
	id integer primary key,
	player_stats_id integer,
	event_half varchar(2),
	event_minute varchar(3),
	event_type varchar(32),
	x1 float,
	y1 float,
	x2 float,
	y2 float
);
`

type League struct {
	Id   string `db:"id"`
	Name string `db:"name"`
}

func main() {
	err := os.Remove("fourfourtwo.db")

	db, err := sqlx.Connect("sqlite3", "fourfourtwo.db")
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.MustExec(schema)

	leagues := []League{
		{"23", "La Liga"},
		{"8", "Premier League"},
		{"21", "Serie A"},
		{"22", "Bundesliga"},
		{"24", "Ligue 1"},
		{"5", "UEFA Champions League"},
	}

	for _, l := range leagues {
		_, err := db.NamedExec(`INSERT INTO league (id, name) VALUES (:id, :name)`, l)
		if err != nil {
			log.Fatal(err)
		}
	}

	checkLeagues := []League{}
	db.Select(&checkLeagues, "SELECT * FROM league")

	for _, l := range checkLeagues {
		fmt.Println(l)
	}
}
