package main

import (
	"database/sql"
	"log"
)

func createRootTreeTable() {
	log.Printf("Creating root_tree table")
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Error starting transaction: %v\n", err)
	}
	defer tx.Rollback()
	_, err = tx.Exec("CREATE TABLE root_tree (id bigserial primary key, created_datetime timestamp NOT NULL DEFAULT now(), tree jsonb NOT NULL);")
	if err != nil {
		log.Fatalf("Error creating table: %v\n", err)
	}
	log.Printf("Inserting initial tree")
	_, err = tx.Exec("INSERT INTO root_tree (tree) VALUES ('{}');")
	if err != nil {
		log.Fatalf("Error inserting initial tree: %v\n", err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatalf("Error committing transaction: %v\n", err)
	}
}

func setupTables() {
	var name string
	err := db.QueryRow("SELECT table_name FROM information_schema.tables WHERE table_name = 'root_tree';").Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			createRootTreeTable()
		} else {
			log.Fatalf("Error getting tables: %v\n", err)
		}
	}
}
