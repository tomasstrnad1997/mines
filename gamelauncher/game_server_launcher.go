package gamelauncher

import (
	"fmt"
	"net"

	"github.com/tomasstrnad1997/mines/protocol"
	"github.com/tomasstrnad1997/mines/server"
)

type matchmakingServer struct {
	controller *protocol.ConnectionController
}

type GameLauncher struct {
	host string
	nextServerId int 
    listener net.Listener
    GameServers map[int] *server.Server
	mmServers map[string]*matchmakingServer
}

func (launcher *GameLauncher) SpawnNewGameServer(name string) (*server.Server, error){
	server, err := server.SpawnServer(launcher.nextServerId, name, 0)
	if err != nil {
		return nil, err
	}
	launcher.GameServers[launcher.nextServerId] = server
	launcher.nextServerId++
	return server, nil
}

func (launcher *GameLauncher) RegisterHandlers(mmServer *matchmakingServer){
    mmServer.controller.RegisterHandler(protocol.SpawnServerRequest, func(bytes []byte) error { 
		var requestId uint32
        name, err := protocol.DecodeSpawnServerRequest(bytes, &requestId)
		if err != nil {
			return err
		}
		server, err := launcher.SpawnNewGameServer(name)
		if err != nil {
			return err
		}
		println(fmt.Sprintf("Spawned server: %s at port %d", name, server.Port))
		info := server.GetServerInfo()
		info.Host = launcher.host
		message, err := protocol.EncodeServerSpawned(info, &requestId)
		if err != nil {
			return err
		}
		if err = mmServer.controller.SendMessage(message); err != nil{
			return err
		}
		return nil

    })
    mmServer.controller.RegisterHandler(protocol.GetGameServers, func(bytes []byte) error { 
		var requestId uint32
        err := protocol.DecodeGetGameServers(bytes, &requestId)
		if err != nil {
			return err
		}
		serverInfos := make([]*protocol.GameServerInfo, len(launcher.GameServers))
		for i, server := range launcher.GameServers {
			serverInfos[i] = server.GetServerInfo()
			serverInfos[i].Host = launcher.host
		}
		payload, err := protocol.EncodeSendGameServers(serverInfos, &requestId)
		if err != nil {
			return err
		}
		if err = mmServer.controller.SendMessage(payload); err != nil{
			return err
		}
		return nil
    })
}

func (launcher *GameLauncher) Loop(){
    defer launcher.listener.Close()
    for {
        conn, err := launcher.listener.Accept()
        if err != nil {
            println(err)
            return
        }
		controller := protocol.CreateConnectionController()
		controller.SetConnection(conn)
		mmServer := &matchmakingServer{controller: controller}
		launcher.mmServers[controller.GetServerAddress()] = mmServer
		launcher.RegisterHandlers(mmServer)
        go controller.ReadServerResponse()
    }
}


func CreateGameLauncher(host string, port uint16) (*GameLauncher, error){
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }
    servers := make(map[int] *server.Server)
	mmServers := make(map[string] *matchmakingServer)
	launcher := &GameLauncher{host: host, nextServerId: 0, listener: listener, GameServers: servers, mmServers: mmServers}
	return launcher, nil

}

