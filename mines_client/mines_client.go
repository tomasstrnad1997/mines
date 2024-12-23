package main

import (
	"fmt"
	"net"
	"os"

	"github.com/tomasstrnad1997/mines"
	protocol "github.com/tomasstrnad1997/mines_protocol"
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
    for {
        reply := make([]byte, 1024)
        _, err := client.Read(reply)
        if err != nil {
            println("Lost connection to server")
            os.Exit(0)
        }
        print(string(reply))
    }
}

func main() {
    client, err := createClient()
    if err != nil {
        println(err.Error())
        os.Exit(1)
    }
    go ReadServerResponse(client)

    for {
        var x, y int
        var flag rune
        flag = 'X'
        n, _ := fmt.Scanf("%d %d %c\n", &x, &y, &flag)
        var move mines.Move
        if n < 2 {
            fmt.Errorf("Incorrect input")
            continue
        }
        if flag == 'f' || flag == 'F' {
            move = mines.Move{X: x, Y: y, Type: mines.Flag}
            
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
