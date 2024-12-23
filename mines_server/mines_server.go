package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/tomasstrnad1997/mines"
	"github.com/tomasstrnad1997/mines_protocol"
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
    for id := range players {
        println(id)
        if players[id].connected {
            fmt.Fprintln(players[id].client, message)
        }
    }
}

func broadcast(data []byte) {
    for id := range players {
        println(id)
        if players[id].connected {
            players[id].client.Write(data)
        }
    }
}

func handleRequest(player *Player){
    reader := bufio.NewReader(player.client)
    clientsMux.Lock()
    clients[player.id] = true
    clientsMux.Unlock()
    fmt.Printf("Player %d connected from %s to %s\n", player.id, player.client.RemoteAddr(), player.client.LocalAddr())
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
        protocol.HandleMessage(message)    


	}
}
func RegisterHandlers(board *mines.Board){

    protocol.RegisterHandler(protocol.MoveCommand, func(bytes []byte) error { 
        move, err := protocol.DecodeMove(bytes)
        if err != nil{
            return err
        }
        board.MakeMove(*move)
        board.Print()
        return nil
    })

}

// IDEA: move UpdatedCell and this function to mines base module
func CreateUpdatedCells(board *mines.Board, cells []*mines.Cell) ([]protocol.UpdatedCell, error){
    updates := make([]protocol.UpdatedCell, len(cells))
    var value byte
    for i, cell := range cells {
        
        if cell.Revealed {
            if cell.Mine {
                value = protocol.ShowMine
            }else {
                value = (byte(mines.GetNumberOfMines(board, cell)))
            }
        } else if cell.Flagged {
            value = protocol.ShowFlag
        } else {
            return nil, fmt.Errorf("Unknown update cell")
        }
        updates[i] = protocol.UpdatedCell{X: cell.X, Y: cell.Y, Value:value}
    }
    return updates, nil
    
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
        go handleRequest(player)
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
