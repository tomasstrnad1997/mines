package main

import (
	"github.com/tomasstrnad1997/mines/matchmaking"
)

func main(){
	server, err := matchmaking.CreateMatchMakingServer(42071)
	if err != nil {
		println("Failed to create matchmaking server")
	}
	server.Run()
	
}
