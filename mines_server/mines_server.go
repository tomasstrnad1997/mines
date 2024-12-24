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


func StartNewGame(width, height, n_mines int) (*mines.Board, error){
    board, err := mines.CreateBoard(width, height, n_mines);
    if err != nil {
        fmt.Println(err)
        return nil, err
    }
    return board, nil
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

func handleRequest(player *Player, board *mines.Board){
    reader := bufio.NewReader(player.client)
    clientsMux.Lock()
    clients[player.id] = true
    clientsMux.Unlock()
    fmt.Printf("Player %d connected from %s to %s\n", player.id, player.client.RemoteAddr(), player.client.LocalAddr())
    sendInitialMessages(player, board)
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
func RegisterHandlers(board *mines.Board){

    protocol.RegisterHandler(protocol.MoveCommand, func(bytes []byte) error { 
        move, err := protocol.DecodeMove(bytes)
        if err != nil{
            return err
        }
        updated, err := board.MakeMove(*move)
        if err != nil {
            return err
        }
        if len(updated.UpdatedCells) > 0 {
            cells, err := mines.CreateUpdatedCells(board, updated.UpdatedCells)
            if err != nil {
                return err
            }
            encoded, err := protocol.EncodeCellUpdates(cells)
            if err != nil{
                return err
            }
            broadcast(encoded)
        }
        
        board.Print()
        return nil
    })

}


func createServer() (net.Listener, error){
    
    listener, err := net.Listen("tcp", "localhost:8080")
    if err != nil {
        fmt.Println("Failed to start server:", err.Error())
        return nil, err
    }
    return listener, nil
}

func RunGame(board *mines.Board){
    id := 0
    server, err := createServer()
    if err != nil {
        return 
    }
    defer server.Close()
    for {
        conn, err := server.Accept()
        if err != nil {
            println(err)
            return
        }
        player := &Player{conn, id, true}
        players[player.id] = player
        go handleRequest(player, board)
        id++
    }


}


func main() {
    board, err := mines.CreateBoard(10, 10, 10);
    if err != nil {
        fmt.Println(err)
        return
    }
    RegisterHandlers(board)
    fmt.Println("Server")
    board.Print()

    RunGame(board)
}
