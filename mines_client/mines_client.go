package main

import (
	"bufio"
	"net"
	"os"
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
    reader := bufio.NewReader(os.Stdin)
    go ReadServerResponse(client)

    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            println(err.Error())
            return
        }
        client.Write([]byte(line))
    }   
}
