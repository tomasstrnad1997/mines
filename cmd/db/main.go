package main

import (
	"log"

	"github.com/tomasstrnad1997/mines/db"
)


func main(){
	store, err := db.InitStore()
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	if err = store.InitializeTables(); err != nil {
		println(err.Error())
		log.Fatalf("Failed to create tables: %v", err)
	}
	log.Println("Tables created")
}
