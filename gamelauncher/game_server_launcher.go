package gamelauncher

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/tomasstrnad1997/mines/protocol"
	"github.com/tomasstrnad1997/mines/server"
)
type MessageHandler func(data []byte, sender net.Conn) error

type command struct {
    message []byte
    sender net.Conn
}

type GameLauncher struct {
	host string
	nextServerId int 
    server net.Listener
    Game_servers map[int] *server.Server
    handlers map[protocol.MessageType]MessageHandler
    messageChannel chan command
}

func (launcher *GameLauncher) SpawnNewGameServer(name string) (*server.Server, error){
	server, err := server.SpawnServer(launcher.nextServerId, name)
	if err != nil {
		return nil, err
	}
	launcher.Game_servers[launcher.nextServerId] = server
	launcher.nextServerId++
	return server, nil
}

func (launcher *GameLauncher) registerHandler(msgType protocol.MessageType, handler MessageHandler){
    launcher.handlers[msgType] = handler
}

func (launcher *GameLauncher) HandleMessage(data []byte, sender net.Conn) error{
    if data == nil {
        return fmt.Errorf("Cannot handle empty message")
    }
    msgType := protocol.MessageType(data[0])
	handler, exists := launcher.handlers[msgType]
	if !exists {
		return fmt.Errorf("No handler registered for message type: %d", msgType)
	}
	return handler(data, sender)
}

func (launcher *GameLauncher) ManageCommands(){
    for command := range launcher.messageChannel{
        err := launcher.HandleMessage(command.message, command.sender)
        if err != nil {
            println(err.Error())
        }

    }
}

func (launcher *GameLauncher) RegisterHandlers(){
    launcher.registerHandler(protocol.SpawnServerRequest, func(bytes []byte, sender net.Conn) error { 
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
		_, err = sender.Write(message)
		if err != nil{
			return err
		}
		return nil

    })
    launcher.registerHandler(protocol.GetGameServers, func(bytes []byte, sender net.Conn) error { 
		var requestId uint32
        err := protocol.DecodeGetGameServers(bytes, &requestId)
		if err != nil {
			return err
		}
		serverInfos := make([]*protocol.GameServerInfo, len(launcher.Game_servers))
		for i, server := range launcher.Game_servers {
			serverInfos[i] = server.GetServerInfo()
			serverInfos[i].Host = launcher.host
		}
		payload, err := protocol.EncodeSendGameServers(serverInfos, &requestId)
		if err != nil {
			return err
		}
		sender.Write(payload)

		return nil
    })
}

func (launcher *GameLauncher) handleRequest (sender net.Conn){
    reader := bufio.NewReader(sender)
    fmt.Printf("Client connected from %s to %s\n", sender.RemoteAddr(), sender.LocalAddr())
	for {
        header := make([]byte, protocol.HeaderLength)
		bytesRead, err := reader.Read(header)
		if err != nil  || bytesRead != protocol.HeaderLength{
            fmt.Printf("Client disconnected \n")
			sender.Close()
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
        launcher.messageChannel <- command{message, sender}
	}
}

func (launcher *GameLauncher) Loop(){
    defer launcher.server.Close()
    for {
        conn, err := launcher.server.Accept()
        if err != nil {
            println(err)
            return
        }
        go launcher.handleRequest(conn)
    }
}


func CreateGameLauncher(host string, port uint16) (*GameLauncher, error){
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }
    servers := make(map[int] *server.Server)
    handlers := make(map[protocol.MessageType]MessageHandler)
    ch := make(chan command)
	launcher := &GameLauncher{host, 0, listener, servers, handlers, ch}

	launcher.RegisterHandlers()
	return launcher, nil

}

func main() {
	launcher, err := CreateGameLauncher("0.0.0.0", 42070)
	if err != nil {
        fmt.Println("Failed to start game launcher server:", err.Error())
		return
	}
	go launcher.ManageCommands()
	launcher.Loop()
	
}
