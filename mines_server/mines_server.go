package server

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/tomasstrnad1997/mines"
	protocol "github.com/tomasstrnad1997/mines_protocol"
)

type Player struct{
    client net.Conn
    id int
    connected bool
}

var (
    clients = make(map[int]bool)
    players = make(map[int]*Player)
    clientsMux sync.Mutex
)

type Game struct {
    board *mines.Board
    parameters mines.GameParams
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
    game *Game
    gameRunning bool
    handlers map[protocol.MessageType]MessageHandler
    messageChannel chan command
	Port uint16
}

func StartNewGame(params mines.GameParams) (*Game, error){
    board, err := mines.CreateBoardFromParams(params)
    if err != nil {
        fmt.Println(err)
        return nil, err
    }
    return &Game{board, params}, nil
}

func (server *Server) StartGame(params mines.GameParams) error{
    game, err := StartNewGame(params)
    if err != nil {
        return err
    }
    server.game = game
    broadcastTextMessage(fmt.Sprintf("Starting a new game...\nNumber of mines %d", params.Mines))

    println("Starting a new game")
    startMsg, err := protocol.EncodeGameStart(params)
    if err != nil {
        return err
    }
    broadcast(startMsg)
    server.gameRunning = true
    return nil
}

func broadcastTextMessage(message string) {
    encoded, err := protocol.EncodeTextMessage(message) 
    if err != nil{
        println("Failed to create message")
        return
    }
    broadcast(encoded)
}

func broadcast(data []byte) {
    for id := range players {
        if players[id].connected {
            players[id].client.Write(data)
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
    player.client.Write(data)
}

func sendInitialMessages(player *Player, server *Server) (error) {
    if server.game.board == nil {
        return nil
    }
    startMsg, err := protocol.EncodeGameStart(server.game.parameters)
    if err != nil {
        return err
    }
    sendMessage(startMsg, player)
    cellUpdates, err := server.game.board.CreateCellUpdates()
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
    clientsMux.Lock()
    clients[player.id] = true
    clientsMux.Unlock()
    fmt.Printf("Player %d connected from %s to %s\n", player.id, player.client.RemoteAddr(), player.client.LocalAddr())
    if server.gameRunning {
        sendInitialMessages(player, server)
    }
    broadcastTextMessage(fmt.Sprintf("Player %d connected from %s to %s", player.id, player.client.RemoteAddr(), player.client.LocalAddr()))
	for {
        header := make([]byte, protocol.HeaderLength)
		bytesRead, err := reader.Read(header)
		if err != nil  || bytesRead != protocol.HeaderLength{
            fmt.Printf("Player %d disconnected \n", player.id)
            broadcastTextMessage(fmt.Sprintf("Player %d disconnected", player.id))
            players[player.id].connected = false
            clientsMux.Lock()
            clients[player.id] = false
            clientsMux.Unlock()
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
            broadcast(msg)
        }
        broadcastTextMessage(fmt.Sprintf("Player %d requested new game", source))
        return server.StartGame(*params)
    })
    server.registerHandler(protocol.MoveCommand, func(bytes []byte, source int) error { 
        if !server.gameRunning  {
            sendTextMessage("Game not running. Cant make moves.", players[source])
            return nil
        }
        board := server.game.board
        move, err := protocol.DecodeMove(bytes)
        if err != nil{
            return err
        }
        moveResult, err := board.MakeMove(*move)
        if err != nil {
            return err
        }
        if len(moveResult.UpdatedCells) > 0 {
            cells, err := mines.CreateUpdatedCells(board, moveResult.UpdatedCells)
            if err != nil {
                return err
            }
            encoded, err := protocol.EncodeCellUpdates(cells)
            if err != nil{
                return err
            }
            broadcast(encoded)
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
            broadcast(endMsg)
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

func createServer(id int, name string) (*Server, error){
    listener, err := net.Listen("tcp", "0.0.0.0:0")
    if err != nil {
        fmt.Println("Failed to start server:", err.Error())
        return nil, err
    }
    messageHandlers := make(map[protocol.MessageType]MessageHandler)
    ch := make(chan command)
	port := listener.Addr().(*net.TCPAddr).Port
    return &Server{id, name, listener, nil, false, messageHandlers, ch, uint16(port)}, nil
}

func serverLoop(server *Server){
    defer server.server.Close()
    id := 0
    for {
        conn, err := server.server.Accept()
        if err != nil {
            println(err)
            return
        }
        player := &Player{conn, id, true}
        players[player.id] = player
        go handleRequest(player, server)
        id++
    }
}

func SpawnServer(id int, name string) (*Server, error){
    server, err := createServer(id, name)
    if err != nil {
        return nil, err
    }
    server.RegisterHandlers()
    go server.manageCommands()
	go serverLoop(server)
	return server, nil
}


func main() {
    SpawnServer(0, "Default Server")
}
