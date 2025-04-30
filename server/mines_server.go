package server

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/tomasstrnad1997/mines/mines"
	"github.com/tomasstrnad1997/mines/protocol"
)

type Player struct{
    client net.Conn
    id int
    connected bool
	writeMutex sync.Mutex
}

type MessageHandler func(data []byte, source int) error

type command struct {
    message []byte
    player *Player
}
type Server struct {
	id int
	Name string
    server net.Listener
    game *mines.Game
    gameRunning bool
    handlers map[protocol.MessageType]MessageHandler
    messageChannel chan command
	Port uint16
    clients map[int]bool
    players map[int]*Player
    clientsMux sync.Mutex
}

func (server *Server) GetNumberOfPlayers() int{
	count := 0
	for _, player := range server.players {
		if player.connected {
			count++
		}
	}
	return count
}

func (server *Server) GetServerInfo() (*protocol.GameServerInfo) {
	return &protocol.GameServerInfo{Name: server.Name, Host: "", Port: server.Port, PlayerCount: server.GetNumberOfPlayers()}
	
}

func (server *Server) StartGame(params mines.GameParams) error{
    game, err := mines.CreateGame(params)
    if err != nil {
        return err
    }
    server.game = game
    server.broadcastTextMessage(fmt.Sprintf("Starting a new game...\nNumber of mines %d", params.Mines))

    println("Starting a new game")
    startMsg, err := protocol.EncodeGameStart(params)
    if err != nil {
        return err
    }
    server.broadcast(startMsg)
    server.gameRunning = true
    return nil
}

func (server *Server) broadcastTextMessage(message string) {
    encoded, err := protocol.EncodeTextMessage(message) 
    if err != nil{
        println("Failed to create message")
        return
    }
    server.broadcast(encoded)
}

func (server *Server) broadcast(data []byte) {
    for _, player := range server.players {
        if player.connected {
			player.writeMutex.Lock()
            player.client.Write(data)
			player.writeMutex.Unlock()
        }
    }
}
func sendTextMessage(msg string, player *Player) {
    encoded, err := protocol.EncodeTextMessage(msg)
    if err != nil {
        println("Failed to create a message")
        return
    }
    sendMessage(encoded, player)
}

func sendMessage(data []byte, player *Player) {
	if player.connected {
		player.writeMutex.Lock()
		player.client.Write(data)
		player.writeMutex.Unlock()
	}
}

func (server *Server) sendInitialMessages(player *Player) (error) {
    startMsg, err := protocol.EncodeGameStart(server.game.Params)
    if err != nil {
        return err
    }
    sendMessage(startMsg, player)
    cellUpdates, err := server.game.GetChangedCellUpdates()
    if err != nil {
        return err
    }
    updateMsg, err := protocol.EncodeCellUpdates(cellUpdates)
    if err != nil {
        return err
    }
    sendMessage(updateMsg, player)

    return nil
}

func handleRequest(player *Player, server *Server){
    reader := bufio.NewReader(player.client)
    server.clientsMux.Lock()
    server.clients[player.id] = true
    server.clientsMux.Unlock()
    fmt.Printf("Player %d connected from %s to %s\n", player.id, player.client.RemoteAddr(), player.client.LocalAddr())
    if server.gameRunning {
        server.sendInitialMessages(player)
    }
    server.broadcastTextMessage(fmt.Sprintf("Player %d connected from %s to %s", player.id, player.client.RemoteAddr(), player.client.LocalAddr()))
	for {
        header := make([]byte, protocol.HeaderLength)
		bytesRead, err := reader.Read(header)
		if err != nil  || bytesRead != protocol.HeaderLength{
            fmt.Printf("Player %d disconnected \n", player.id)
            server.broadcastTextMessage(fmt.Sprintf("Player %d disconnected", player.id))
            server.players[player.id].connected = false
            server.clientsMux.Lock()
            server.clients[player.id] = false
            server.clientsMux.Unlock()
			player.client.Close()
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
        server.messageChannel <- command{message, player}
	}
}

func (server *Server) HandleMessage(data []byte, source int) error{
    if data == nil {
        return fmt.Errorf("Cannot handle empty message")
    }
    msgType := protocol.MessageType(data[0])
	handler, exists := server.handlers[msgType]
	if !exists {
		return fmt.Errorf("No handler registered for message type: %d", msgType)
	}
	return handler(data, source)
}

func (server *Server) registerHandler(msgType protocol.MessageType, handler MessageHandler){
    server.handlers[msgType] = handler
}

func (server *Server) RegisterHandlers(){
    server.registerHandler(protocol.StartGame, func(bytes []byte, source int) error { 
        params, err := protocol.DecodeGameStart(bytes)
        if err != nil {
            return err
        }
        if server.gameRunning {
            msg, err := protocol.EncodeGameEnd(protocol.Aborted)
            if err != nil {
                return err
            }
            server.broadcast(msg)
        }
        server.broadcastTextMessage(fmt.Sprintf("Player %d requested new game", source))
        return server.StartGame(*params)
    })
    server.registerHandler(protocol.MoveCommand, func(bytes []byte, source int) error { 
        if !server.gameRunning  {
            sendTextMessage("Game not running. Cant make moves.", server.players[source])
            return nil
        }
        move, err := protocol.DecodeMove(bytes)
        if err != nil{
            return err
        }
		move.PlayerId = source
        moveResult, gamemodeInfo, err := server.game.MakeMove(*move)
        if err != nil {
            return err
        }
        if len(moveResult.UpdatedCells) > 0 {
            cells, err := server.game.CreateCellUpdates(moveResult.UpdatedCells)
            if err != nil {
                return err
            }
            encoded, err := protocol.EncodeCellUpdates(cells)
            if err != nil{
                return err
            }
            server.broadcast(encoded)
			if gamemodeInfo != nil {
				encoded, err = protocol.EncodeGamemodeInfo(gamemodeInfo)
				if err != nil{
					return err
				}
            	server.broadcast(encoded)
			}
        }
        var endMsg []byte
        switch moveResult.Result {
        case mines.MineBlown:
            endMsg, err = protocol.EncodeGameEnd(protocol.Loss)
        case mines.GameWon:
            endMsg, err = protocol.EncodeGameEnd(protocol.Win)
        default:
            endMsg, err = nil, nil
        }
        if err != nil {
            return err
        }
        if endMsg != nil {
            server.broadcast(endMsg)
            server.gameRunning = false
        }
        return nil
    })
}

func (server *Server) manageCommands(){
    for command := range server.messageChannel{
        err := server.HandleMessage(command.message, command.player.id)
        if err != nil {
            println(err.Error())
        }

    }
}

func createServer(id int, name string, port uint16) (*Server, error){
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
    if err != nil {
        fmt.Println("Failed to start server:", err.Error())
        return nil, err
    }
    handlers := make(map[protocol.MessageType]MessageHandler)
    messageChannel := make(chan command)
	serverPort := listener.Addr().(*net.TCPAddr).Port
	clients := make(map[int]bool)
	players := make(map[int]*Player)
    server := &Server{
        id:             id,
        Name:           name,
        server:         listener,
        game:           nil,
        gameRunning:    false,
        handlers:       handlers,
        messageChannel: messageChannel,
        Port:           uint16(serverPort),
        clients:        clients,
        players:        players,
    }
    return server, nil
}

func serverLoop(server *Server){
    defer server.server.Close()
    id := 1
    for {
        conn, err := server.server.Accept()
        if err != nil {
            println(err)
            return
        }
        player := &Player{
			id: id,
			client: conn,
			connected: true,
		}
        server.players[player.id] = player
        go handleRequest(player, server)
        id++
    }
}

func SpawnServer(id int, name string, port uint16) (*Server, error){
    server, err := createServer(id, name, port)
    if err != nil {
        return nil, err
    }
    server.RegisterHandlers()
    go server.manageCommands()
	go serverLoop(server)
	return server, nil
}

