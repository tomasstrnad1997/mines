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

func setupMMserverAndLauncher(mmPort, launcherPort uint16) (*matchmaking.MatchmakingServer, *gamelauncher.GameLauncher, error){

	mmServer, err := matchmaking.CreateMatchMakingServer(mmPort)
	if err != nil {
		return nil, nil, err
	}
	go mmServer.Run()

	// Start game launcher and connect to it from mm server
	launcher, err := gamelauncher.CreateGameLauncher("localhost", launcherPort)
	if err != nil {
		return nil, nil, err
	}
	go launcher.ManageCommands()
	go launcher.Loop()
	mmServer.ConnectToLauncher("localhost", launcherPort)
	return mmServer, launcher, nil
}

func waitForResponse(conn net.Conn, t *testing.T) ([]byte){
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
	return message
}

func TestGetServerList(t *testing.T){
	mmPort := uint16(42075)
	launcherPort := uint16(42076)
	nServers := 5
	_, launcher, err := setupMMserverAndLauncher(mmPort, launcherPort)
	if err != nil{
		t.Fatalf("Setup failed %v", err)
	}
	for i := range nServers {
		launcher.SpawnNewGameServer(fmt.Sprintf("Server %d", i))
	}


	// Connect to matchmaking server as a player
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", mmPort))
	if err != nil {
		t.Fatalf("Cannot connect to game launcher: %v", err)
	}
	defer conn.Close()
	// Request a server spawn
	payload, err := protocol.EncodeGetGameServers(nil)
	conn.Write(payload)
	// Wait for response from MM server
	message := waitForResponse(conn, t)
	infos, err := protocol.DecodeSendGameServers(message, nil)
	if err != nil {
		t.Fatalf("Failed decode server info message")
	}
	for i := range(nServers) {
		name :=	fmt.Sprintf("Server %d", i)
		found := false
		for _, server := range infos{
			if server.Name == name {
				found = true
				continue
			}
		}
		if !found {
			t.Fatalf("Server %s not running", name)
		}
	}

}

func TestGameServerSpawn(t *testing.T){
	mmPort := uint16(42071)
	launcherPort := uint16(42070)
	_, _, err := setupMMserverAndLauncher(mmPort, launcherPort)
	if err != nil{
		t.Fatalf("Setup failed %v", err)
	}

	serverName := "Testing server"

	// Connect to matchmaking server as a player
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", mmPort))
	if err != nil {
		t.Fatalf("Cannot connect to game launcher: %v", err)
	}
	defer conn.Close()
	
	// Request a server spawn
	payload, err := protocol.EncodeSpawnServerRequest(serverName, nil)
	conn.Write(payload)
	// Wait for response from MM server
	message := waitForResponse(conn, t)
	serverInfo, err := protocol.DecodeServerSpawned(message, nil)
	if err != nil {
		t.Fatalf("Failed decode server info message")
	}
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
