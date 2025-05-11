package matchmaking

import (
	"fmt"
	"net"
	"sync"

	"github.com/tomasstrnad1997/mines/db"
	"github.com/tomasstrnad1997/mines/players"
	"github.com/tomasstrnad1997/mines/protocol"
)

type command struct {
    message []byte
    sender net.Conn
}

type Player struct{
	controller *protocol.ConnectionController
}

type GameLauncher struct {
	controller *protocol.ConnectionController
}

type MatchmakingServer struct{
	GameLaunchers map[string] *GameLauncher
	listener net.Listener
    messageChannel chan command
	Players map[string]*Player
	pendingRequests sync.Map
	currentRequestId uint32
	requestIdMux sync.Mutex
	db *db.SQLStore
	PlayerService *players.Service
}

func (server *MatchmakingServer) GetNextRequestId() uint32{
	server.requestIdMux.Lock()
	defer server.requestIdMux.Unlock()
	requestId := server.currentRequestId
	server.currentRequestId++
	return requestId
}

func (server *MatchmakingServer) RegisterPlayerHandlers(player *Player){
    player.controller.RegisterHandler(protocol.SpawnServerRequest, func(bytes []byte) error { 
		launcher, err := server.chooseGameLauncher()
		if err != nil {
			return err
		}
		serverName, err := protocol.DecodeSpawnServerRequest(bytes, nil)
		if err != nil {
			return err
		}
		requestId := server.GetNextRequestId()
		payload, err := protocol.EncodeSpawnServerRequest(serverName, &requestId)
		if err != nil {
			return err
		}
		server.pendingRequests.Store(requestId, player)
		launcher.controller.SendMessage(payload)
		return nil
    })
    player.controller.RegisterHandler(protocol.GetGameServers, func(bytes []byte) error { 
        err := protocol.DecodeGetGameServers(bytes, nil)
		if err != nil {
			return err
		}
		// Request all servers from all launchers
		for _, launcher := range server.GameLaunchers {
			requestId := server.GetNextRequestId()
			payload, err := protocol.EncodeGetGameServers(&requestId)
			if err != nil {
				return err
			}
			server.pendingRequests.Store(requestId, player)
			launcher.controller.SendMessage(payload)
		}
		return nil
    })
    player.controller.RegisterHandler(protocol.RegisterPlayerRequest, func(bytes []byte) error { 
		playerData, err := protocol.DecodeRegisterPlayerRequest(bytes)
		if err != nil {
			return err
		}
		if err := server.PlayerService.Register(playerData.Name, playerData.Password);err != nil {
			return err
		}
		return nil
    })
}

func (server *MatchmakingServer) RegisterLauncherHandlers(launcher *GameLauncher){
    launcher.controller.RegisterHandler(protocol.ServerSpawned, func(bytes []byte) error { 
		var requestId uint32
		info, err := protocol.DecodeServerSpawned(bytes, &requestId)
		if err != nil {
			return err
		}
		value, ok := server.pendingRequests.LoadAndDelete(requestId)
		if !ok {
			return fmt.Errorf("Request id is not in pending requests")
		}
		// Do checks if the client is still connected
		player, ok := value.(*Player)
		if !ok {
			return fmt.Errorf("Stored request is not *Player")
		}
		payload, err := protocol.EncodeServerSpawned(info, nil)
		if err != nil {
			return err
		}
		player.controller.SendMessage(payload)
		return nil
    })
    launcher.controller.RegisterHandler(protocol.SendGameServers, func(bytes []byte) error { 
		var requestId uint32
		infos, err := protocol.DecodeSendGameServers(bytes, &requestId)
		if err != nil {
			return err
		}
		value, ok := server.pendingRequests.LoadAndDelete(requestId)
		if !ok {
			return fmt.Errorf("Request id is not in pending requests")
		}
		// Do checks if the client is still connected
		player, ok := value.(*Player)
		if !ok {
			return fmt.Errorf("Stored request is not *Player")
		}
		payload, err := protocol.EncodeSendGameServers(infos, nil)
		if err != nil {
			return err
		}
		player.controller.SendMessage(payload)
		return nil
    })
}

func (server *MatchmakingServer) chooseGameLauncher() (*GameLauncher, error){
	for _, value := range server.GameLaunchers{
		return value, nil
	}
	return nil, fmt.Errorf("Not game launchers available")
}

func (server *MatchmakingServer) Run(){
    defer server.listener.Close()
    for {
        conn, err := server.listener.Accept()
        if err != nil {
            println(err)
            return
        }
		controller := protocol.CreateConnectionController()
		controller.SetConnection(conn)
		player := &Player{controller: controller}
		server.RegisterPlayerHandlers(player)
		go player.controller.ReadServerResponse()
    }
}

func (server *MatchmakingServer) ConnectToLauncher(host string, port uint16, reconnect bool) error{
	controller := protocol.CreateConnectionController()
	controller.AttemptReconnect = reconnect
	if err := controller.Connect(host, port); err != nil {
		return err
	}
	launcher := &GameLauncher{controller: controller}
	server.GameLaunchers[controller.GetServerAddress()] = launcher
	server.RegisterLauncherHandlers(launcher)
	go launcher.controller.ReadServerResponse()
	return nil
}

func CreateMatchMakingServer(port uint16) (*MatchmakingServer, error){
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }
    launchers := make(map[string] *GameLauncher)
    plrs := make(map[string] *Player)
	store, err := db.InitStore()
	if err != nil {
		return nil, err
	}
	pService := &players.Service{Store: store}
	
    ch := make(chan command)
	server := &MatchmakingServer{listener: listener, messageChannel: ch, GameLaunchers: launchers, Players: plrs, db: store, PlayerService: pService}
	return server, nil
}
