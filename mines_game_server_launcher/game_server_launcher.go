package main

import (
	"fmt"
	"net"

	server "github.com/tomasstrnad1997/mines_server"
)
type GameLauncher struct {
	nextServerId int 
    server net.Listener
    servers map[int] *server.Server
}

func (mmServer *GameLauncher) SpawnNewGameServer(name string) (*server.Server, error){
	server, err := server.SpawnServer(mmServer.nextServerId, name)
	if err != nil {
		return nil, err
	}
	mmServer.servers[mmServer.nextServerId] = server
	mmServer.nextServerId++
	return server, nil
}

func CreateGameLauncher(port uint16) (*GameLauncher, error){
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }
    servers := make(map[int] *server.Server)
	return &GameLauncher{0, listener, servers}, nil
}

func main() {
	launcher, err := CreateGameLauncher(42070)
	if err != nil {
        fmt.Println("Failed to start game launcher server:", err.Error())
		return
	}
	for i := range 3{
		server, _ := launcher.SpawnNewGameServer(fmt.Sprintf("Game server %d", i))
		println(server.Port)
		
	}
	for {
	}
	
}
