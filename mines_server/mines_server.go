package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	"github.com/tomasstrnad1997/mines"
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


func StartNewGame(width, heigt, n_mines int) (*mines.Board, error){
    board, err := mines.CreateBoard(10, 10, 5);
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

func handleRequest(player *Player, board *mines.Board){
    reader := bufio.NewReader(player.client)
    clientsMux.Lock()
    clients[player.id] = true
    clientsMux.Unlock()
    fmt.Printf("Player %d connected from %s to %s\n", player.id, player.client.RemoteAddr(), player.client.LocalAddr())
    broadcastMessage(fmt.Sprintf("Player %d connected from %s to %s", player.id, player.client.RemoteAddr(), player.client.LocalAddr()))

	for {
		message, err := reader.ReadString('\n')
		if err != nil {

            fmt.Printf("Player %d disconnected \n", player.id)
            broadcastMessage(fmt.Sprintf("Player %d disconnected", player.id))
            players[player.id].connected = false
            clientsMux.Lock()
            clients[player.id] = false
            clientsMux.Unlock()
			player.client.Close()
			return
		}
        boardMutex.Lock()
        result, err := board.ProcessTextCommand(string(message))
        if err != nil {
            fmt.Fprintln(player.client, err.Error())
        }else{
            println(len(result.UpdatedCells))
            if len(result.UpdatedCells) > 0 {
                broadcastMessage(fmt.Sprintf("Player %d played: %s Updated cells: %d", player.id, string(message), len(result.UpdatedCells)))
            }
        }
        boardMutex.Unlock()
        fmt.Printf("Player %d played: %s\n", player.id, string(message))

        board.Print()
	}
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
    fmt.Println("Server")
    board.Print()

    RunGame(board)
}
