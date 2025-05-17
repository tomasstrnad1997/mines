package main

import (
	"fmt"
	"os"

	"github.com/tomasstrnad1997/mines/matchmaking"
)

func main(){
	os.Setenv("DB_PATH", "../../var/data.db")
	server, err := matchmaking.CreateMatchMakingServer(42071)
	if err != nil {
		fmt.Printf("Failed to create matchmaking server: %v\n", err)
		return
	}
	go server.Run()
	err = server.ConnectToLauncher("localhost", 42070, true)
	if err != nil {
		fmt.Println(err.Error())
	}
	for {}
	
}
