package main

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
    boardMutex sync.RWMutex
)

type Game struct {
    board *mines.Board
    parameters mines.GameParams
}

type Server struct {
    server net.Listener
    game *Game
    gameRunning bool
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
    broadcastMessage("Starting a new game...")
    println("Starting a new game")
    startMsg, err := protocol.EncodeGameStart(params)
    if err != nil {
        return err
    }
    broadcast(startMsg)
    server.gameRunning = true
    return nil
}

func broadcastMessage(message string) {
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

func sendMessage(data []byte, player *Player) {
    player.client.Write(data)
}

func sendInitialMessages(player *Player, board *mines.Board) (error) {
    if board == nil {
        return nil
    }
    startMsg, err := protocol.EncodeGameStart(mines.GameParams{Width: board.Width, Height: board.Height, Mines: board.Mines})
    if err != nil {
        return err
    }
    sendMessage(startMsg, player)
    cellUpdates, err := board.CreateCellUpdates()
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
        sendInitialMessages(player, server.game.board)
    }
    broadcastMessage(fmt.Sprintf("Player %d connected from %s to %s", player.id, player.client.RemoteAddr(), player.client.LocalAddr()))
	for {
        header := make([]byte, 4)
		bytesRead, err := reader.Read(header)
		if err != nil  || bytesRead != 4{
            fmt.Printf("Player %d disconnected \n", player.id)
            broadcastMessage(fmt.Sprintf("Player %d disconnected", player.id))
            players[player.id].connected = false
            clientsMux.Lock()
            clients[player.id] = false
            clientsMux.Unlock()
			player.client.Close()
			return
		}
        messageLenght := int(binary.BigEndian.Uint16(header[2:4]))
        message := make([]byte, messageLenght+4)
        copy(message[0:4], header)
        _, err = io.ReadFull(reader, message[4:])
        if err != nil {
            fmt.Printf("Error reading message")
            continue
        }
        err = protocol.HandleMessage(message)    
        if err != nil {
            println(err.Error())
        }


	}
}
func RegisterHandlers(server *Server){
    protocol.RegisterHandler(protocol.StartGame, func(bytes []byte) error { 
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
        return server.StartGame(*params)
    })
    protocol.RegisterHandler(protocol.MoveCommand, func(bytes []byte) error { 
        if !server.gameRunning  {
            return fmt.Errorf("Game is not running. Cannot accept moves")
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
        board.Print()
        return nil
    })
}

func createServer() (*Server, error){
    listener, err := net.Listen("tcp", "localhost:8080")
    if err != nil {
        fmt.Println("Failed to start server:", err.Error())
        return nil, err
    }
    return &Server{listener, nil, false}, nil
}

func RunGame(){
    id := 0
    server, err := createServer()
    if err != nil {
        return 
    }
    RegisterHandlers(server)
    fmt.Println("Server is running...")
    defer server.server.Close()
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


func main() {
    RunGame()
}
