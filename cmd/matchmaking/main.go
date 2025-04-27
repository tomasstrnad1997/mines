package main

import (
	"fmt"

	"github.com/tomasstrnad1997/mines/matchmaking"
)

func main(){
	server, err := matchmaking.CreateMatchMakingServer(42071)
	if err != nil {
		println("Failed to create matchmaking server")
	}
	go server.Run()
	_, err = server.ConnectToLauncher("localhost", 42070)
	if err != nil {
		fmt.Println(err.Error())
	}
	for {}
	
}
