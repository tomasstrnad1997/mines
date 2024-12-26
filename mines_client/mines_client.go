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
		if err != nil  || bytesRead != 4{
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
        protocol.HandleMessage(message)    
    }
    
}

func createBoard(params mines.GameParams) *[][]rune {
    board := make([][]rune, params.Width)
    for i := range board {
        board[i] = make([]rune, params.Height)
        for j := 0; j < params.Height; j++{
            board[i][j] = '#'
        }
    }
    return &board
}

func printBoard(board *[][]rune){
    print("X")
    for i:=0; i < 10; i++{
        print(i % 10)
    }
    println()
    for y := 0; y < 10; y++{
        print(y % 10)
        for x := 0; x < 10; x++{
            fmt.Printf("%c",(*board)[x][y])
        }
        println()
    }
}

func RegisterHandlers(){
    var board *[][]rune 
    protocol.RegisterHandler(protocol.TextMessage, func(bytes []byte) error { 
        msg, err := protocol.DecodeTextMessage(bytes)
        if err != nil{
            return err
        }
        println(msg)
        return nil     
    })
    protocol.RegisterHandler(protocol.StartGame, func(bytes []byte) error { 
        params, err := protocol.DecodeGameStart(bytes)
        if err != nil{
            return err
        }
        board = createBoard(*params)
        printBoard(board)
        return nil     
    })
    protocol.RegisterHandler(protocol.CellUpdate, func(bytes []byte) error { 
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
            }else{
                rep = '?'
            }
            (*board)[cell.X][cell.Y] = rep
            // fmt.Printf("Move: %d, %d - %c\n", cell.X, cell.Y, rep)
        }
        printBoard(board)
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
