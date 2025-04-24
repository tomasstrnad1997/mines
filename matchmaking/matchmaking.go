package matchmaking

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/tomasstrnad1997/mines/protocol"
)

type MessageHandler func(data []byte, sender net.Conn) error

type command struct {
    message []byte
    sender net.Conn
}

type Player struct{
	host string	
	port uint16
	client net.Conn
	connected bool
}

type GameLauncher struct {
	connection net.Conn
}

type MatchmakingServer struct{
	GameLaunchers map[string] *GameLauncher
    server net.Listener
    handlers map[protocol.MessageType]MessageHandler
    messageChannel chan command
	Players map[string]*Player
	pendingRequests sync.Map
	currentRequestId uint32
	requestIdMux sync.Mutex
}

func (server *MatchmakingServer) GetNextRequestId() uint32{
	server.requestIdMux.Lock()
	defer server.requestIdMux.Unlock()
	requestId := server.currentRequestId
	server.currentRequestId++
	return requestId
}

func (server *MatchmakingServer) registerHandler(msgType protocol.MessageType, handler MessageHandler){
    server.handlers[msgType] = handler
}

func (server *MatchmakingServer) RegisterHandlers(){
    server.registerHandler(protocol.SpawnServerRequest, func(bytes []byte, sender net.Conn) error { 
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
		server.pendingRequests.Store(requestId, sender)
		launcher.connection.Write(payload)
		return nil
    })
    server.registerHandler(protocol.ServerSpawned, func(bytes []byte, sender net.Conn) error { 
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
		requester := value.(net.Conn)
		payload, err := protocol.EncodeServerSpawned(info, nil)
		if err != nil {
			return err
		}
		requester.Write(payload)
		return nil
    })
    server.registerHandler(protocol.GetGameServers, func(bytes []byte, sender net.Conn) error { 
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
			server.pendingRequests.Store(requestId, sender)
			launcher.connection.Write(payload)
		}
		return nil
    })
    server.registerHandler(protocol.SendGameServers, func(bytes []byte, sender net.Conn) error { 
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
		requester := value.(net.Conn)
		payload, err := protocol.EncodeSendGameServers(infos, nil)
		if err != nil {
			return err
		}
		requester.Write(payload)
		return nil
    })
}

func (server *MatchmakingServer) chooseGameLauncher() (*GameLauncher, error){
	for _, value := range server.GameLaunchers{
		return value, nil
	}
	return nil, fmt.Errorf("Not game launchers available")
}

func handlePlayerConnection(player *Player, server *MatchmakingServer){
    reader := bufio.NewReader(player.client)
	addr := player.client.RemoteAddr().String()
    fmt.Printf("Player connected from %s\n", addr)
	for {
        header := make([]byte, protocol.HeaderLength)
		bytesRead, err := reader.Read(header)
		if err != nil  || bytesRead != protocol.HeaderLength{
            fmt.Printf("Player %s disconnected \n", addr)
            server.Players[addr].connected = false
			server.Players[addr].client.Close()
			return
		}
        messageLenght := int(binary.BigEndian.Uint32(header[2:protocol.HeaderLength]))
        message := make([]byte, messageLenght+protocol.HeaderLength)
        copy(message[0:protocol.HeaderLength], header)
        _, err = io.ReadFull(reader, message[protocol.HeaderLength:])
        if err != nil {
            fmt.Printf("Error reading message")
            continue
        }
        server.messageChannel <- command{message, player.client}
	}
}

func (server *MatchmakingServer) handleLauncherConnection(launcher *GameLauncher) error{
    reader := bufio.NewReader(launcher.connection)
    for {
        header := make([]byte, protocol.HeaderLength)
		bytesRead, err := reader.Read(header)
        if err != nil {
            return fmt.Errorf("Lost connection to server\n")
        }
		if bytesRead != protocol.HeaderLength{
            return fmt.Errorf("Failed to read message\n")
		}
        messageLenght := int(binary.BigEndian.Uint32(header[2:protocol.HeaderLength]))
        message := make([]byte, messageLenght+protocol.HeaderLength)
        copy(message[0:protocol.HeaderLength], header)
        _, err = io.ReadFull(reader, message[protocol.HeaderLength:])
        if err != nil {
            return err
        }
        err = server.HandleMessage(message, launcher.connection)    
        if err != nil {
            println(err.Error())
        }
    }
    
}



func (server *MatchmakingServer) Run(){
    defer server.server.Close()
	go server.ManageCommands()
    for {
        conn, err := server.server.Accept()
        if err != nil {
            println(err)
            return
        }
		addr := conn.RemoteAddr().(*net.TCPAddr)
		ip := addr.IP.String()
		port := uint16(addr.Port)
		player := &Player{host:ip, port: port, connected: true, client:conn}
        go handlePlayerConnection(player, server)
    }
}

func (server *MatchmakingServer) HandleMessage(data []byte, sender net.Conn) error{
    if data == nil {
        return fmt.Errorf("Cannot handle empty message")
    }
    msgType := protocol.MessageType(data[0])
	handler, exists := server.handlers[msgType]
	if !exists {
		return fmt.Errorf("No handler registered for message type: %d", msgType)
	}
	return handler(data, sender)
}

func (server *MatchmakingServer) ManageCommands(){
    for command := range server.messageChannel{
        err := server.HandleMessage(command.message, command.sender)
        if err != nil {
            println(err.Error())
        }

    }
}

func (server *MatchmakingServer) ConnectToLauncher(host string, port uint16) (*net.TCPConn, error){
	servAddr := fmt.Sprintf("%s:%d", host, port)
    tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
    if err != nil {
        println("Reslove tpc failed:")
        return nil, err
    }
    conn, err := net.DialTCP("tcp", nil, tcpAddr)
    if err != nil {
        println("Dial failed:")
        return nil, err
    }
	launcher := &GameLauncher{conn}
	server.GameLaunchers[servAddr] = launcher
	go server.handleLauncherConnection(launcher)
    return conn, nil
}

func CreateMatchMakingServer(port uint16) (*MatchmakingServer, error){
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }
    launchers := make(map[string] *GameLauncher)
    players := make(map[string] *Player)
    handlers := make(map[protocol.MessageType]MessageHandler)
    ch := make(chan command)
	server := &MatchmakingServer{server: listener, messageChannel: ch, handlers: handlers, GameLaunchers: launchers, Players: players}
	server.RegisterHandlers()
	return server, nil
}
