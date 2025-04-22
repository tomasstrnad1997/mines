package matchmaking_test

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/tomasstrnad1997/mines/gamelauncher"
	"github.com/tomasstrnad1997/mines/matchmaking"
	"github.com/tomasstrnad1997/mines/protocol"
)

func TestGameServerSpawn(t *testing.T){
	mmPort := uint16(42071)
	launcherPort := uint16(42070)
	serverName := "Testing server"
	mmServer, err := matchmaking.CreateMatchMakingServer(mmPort)
	if err != nil {
		t.Logf("Matchmaking server did not start: %v", err)
	}
	go mmServer.Run()

	// Start game launcher and connect to it from mm server
	launcher, err := gamelauncher.CreateGameLauncher("localhost", launcherPort)
	if err != nil {
		t.Logf("Launcher did not start: %v", err)
	}
	go launcher.ManageCommands()
	go launcher.Loop()

	mmServer.ConnectToLauncher("localhost", launcherPort)

	// Connect to matchmaking server as a player
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", mmPort))
	if err != nil {
		t.Fatalf("Cannot connect to game launcher: %v", err)
	}
	defer conn.Close()
	
	// Request a server spawn
	payload, err := protocol.EncodePlayerSpawnServerRequest(serverName)
	conn.Write(payload)
	
	// Wait for response from MM server
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
    reader := bufio.NewReader(conn)
	header := make([]byte, protocol.HeaderLength)
	bytesRead, err := reader.Read(header)
	if err != nil {
		t.Fatalf("Lost connection to server: %v", err)
	}
	if bytesRead != protocol.HeaderLength{
		t.Fatalf("Failed to read header: %v", err)
	}
	messageLenght := int(binary.BigEndian.Uint32(header[2:protocol.HeaderLength]))
	message := make([]byte, messageLenght+protocol.HeaderLength)
	copy(message[0:protocol.HeaderLength], header)
	_, err = io.ReadFull(reader, message[protocol.HeaderLength:])
	if err != nil {
		t.Fatalf("Failed to read message %v", err)
	}
	infos, err := protocol.DecodeSendGameServers(message)
	if len(infos) != 1 {
		t.Fatalf("Length of recieved servers != 1")
	}
	serverInfo := infos[0]
	if serverInfo.Name != serverName {
		t.Fatalf("Server name mismatch")
	}

	// Try to connect to game server

	gameConn, err := net.Dial("tcp", fmt.Sprintf("[%s]:%d", serverInfo.Host, +serverInfo.Port))
	if err != nil {
		t.Fatalf("Cannot connect to game server: %v", err)
	}
	defer gameConn.Close()
	// encoded, err := protocol.EncodeGameStart(mines.GameParams{Width: 10, Height: 10, Mines: 10})
	// gameConn.Write(encoded)
	



}
