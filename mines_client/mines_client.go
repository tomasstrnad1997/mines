package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/tomasstrnad1997/mines"
	"github.com/tomasstrnad1997/mines_protocol"
)

type BoardView struct {
    board *[][]rune
    params mines.GameParams
}

type MessageHandler func([]byte) error
var messageHandlers = make(map[protocol.MessageType]MessageHandler)
func HandleMessage(bytes []byte) error {

    msgType := protocol.MessageType(bytes[0])
	handler, exists := messageHandlers[msgType]
	if !exists {
		return fmt.Errorf("No handler registered for message type: %d", msgType)
	}
	return handler(bytes)
}

func RegisterHandler(msgType protocol.MessageType, handler MessageHandler) {
	messageHandlers[msgType] = handler
}
func createClient() (*net.TCPConn, error){
    servAddr := "localhost:8080"
    tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
    if err != nil {
        println("Reslove tpc failed:")
        return nil, err
    }
    conn, err := net.DialTCP("tcp", nil, tcpAddr)
    println("RESOLVED TCP")
    if err != nil {
        println("Dial failed:")
        return nil, err
    }
    println("DIALED TCP")
    return conn, nil
}

func ReadServerResponse(client net.Conn){
    reader := bufio.NewReader(client)
    for {
        header := make([]byte, 4)
		bytesRead, err := reader.Read(header)
        if err != nil {
            fmt.Printf("Lost connection to server\n")
            os.Exit(0)
        }
		if bytesRead != 4{
            fmt.Printf("Failed to read message\n")
            os.Exit(0)
		}
        messageLenght := int(binary.BigEndian.Uint16(header[2:4]))
        message := make([]byte, messageLenght+4)
        copy(message[0:4], header)
        _, err = io.ReadFull(reader, message[4:])
        if err != nil {
            fmt.Printf("Error reading message\n")
            continue
        }
        HandleMessage(message)    
    }
    
}

func createBoard(params mines.GameParams) *BoardView {
    board := make([][]rune, params.Width)
    for i := range board {
        board[i] = make([]rune, params.Height)
        for j := 0; j < params.Height; j++{
            board[i][j] = '#'
        }
    }
    return &BoardView{&board, params}
}

func (board *BoardView) Print(){
    print("X")
    for i:=0; i < 10; i++{
        print(i % 10)
    }
    println()
    for y := 0; y < board.params.Height; y++{
        print(y % 10)
        for x := 0; x < board.params.Width; x++{
            fmt.Printf("%c",(*board.board)[x][y])
        }
        println()
    }
}

func RegisterHandlers(){
    var board *BoardView 
    RegisterHandler(protocol.GameEnd, func(bytes []byte) error { 
        endType, err := protocol.DecodeGameEnd(bytes)
        if err != nil {
            return err
        }
        switch endType {
        case protocol.Win:
            println("Game won")
        case protocol.Loss:
            println("Game lost")
        case protocol.Aborted:
            println("Game aborted")
        }
        board = nil
        return nil
    })
    RegisterHandler(protocol.TextMessage, func(bytes []byte) error { 
        msg, err := protocol.DecodeTextMessage(bytes)
        if err != nil{
            return err
        }
        println(msg)
        return nil     
    })
    RegisterHandler(protocol.StartGame, func(bytes []byte) error { 
        params, err := protocol.DecodeGameStart(bytes)
        if err != nil{
            return err
        }
        board = createBoard(*params)
        board.Print()
        return nil     
    })
    RegisterHandler(protocol.CellUpdate, func(bytes []byte) error { 
        updates, err := protocol.DecodeCellUpdates(bytes)
        if err != nil{
            return err
        }
        for _, cell := range updates {
            var rep rune 
            if (cell.Value & 0xF0) == 0{
                rep = rune(cell.Value) + '0'
            }else if cell.Value == mines.ShowFlag{
                rep = 'F'
            }else if cell.Value == mines.ShowMine{
                rep = 'X'
            }else if cell.Value == mines.Unflag{
                rep = '#'
            }else {
                rep = '?'
            }
            (*board.board)[cell.X][cell.Y] = rep
            // fmt.Printf("Move: %d, %d - %c\n", cell.X, cell.Y, rep)
        }
        board.Print()
        return nil
    })

}

func main() {
    client, err := createClient()

    if err != nil {
        println(err.Error())
        os.Exit(1)
    }
    RegisterHandlers()
    go ReadServerResponse(client)

    for {
        var x, y int
        var flag rune
        flag = 'X'
        n, _ := fmt.Scanf("%d %d %c\n", &x, &y, &flag)
        var move mines.Move
        if n < 2 {
            fmt.Println("Incorrect input")
            continue
        }
        if flag == 'f' || flag == 'F' {
            move = mines.Move{X: x, Y: y, Type: mines.Flag}
            
        }else if flag == 'G' {
            encoded, err := protocol.EncodeGameStart(mines.GameParams{Width: x, Height: x, Mines: y})
            if err != nil {
                println(err.Error())
                continue
            }
            client.Write(encoded)
            continue
        }else{
            move = mines.Move{X: x, Y: y, Type: mines.Reveal}
        }
        encoded, err := protocol.EncodeMove(move)
        if err != nil {
            println(err.Error())
            continue
        }
        client.Write(encoded)
    }   
}
