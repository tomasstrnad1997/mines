package main

import (
	"fmt"
	"net"

	server "github.com/tomasstrnad1997/mines_server"
)
type MatchmakingServer struct {
	nextServerId int 
    server net.Listener
    servers map[int] *server.Server
}

func (mmServer *MatchmakingServer) SpawnNewGameServer() (*server.Server, error){
	server, err := server.SpawnServer(mmServer.nextServerId)
	if err != nil {
		return nil, err
	}
	mmServer.servers[mmServer.nextServerId] = server
	mmServer.nextServerId++
	return server, nil
}

func CreateMMServer(port uint16) (*MatchmakingServer, error){
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }
    servers := make(map[int] *server.Server)
	return &MatchmakingServer{0, listener, servers}, nil
}

func main() {
	mmServer, err := CreateMMServer(42069)
	if err != nil {
        fmt.Println("Failed to start matchmaking server:", err.Error())
		return
	}
	for range 3{
		server, _ := mmServer.SpawnNewGameServer()
		println(server.Port)
		
	}
	for {
	}
	
}
