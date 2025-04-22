package gamelauncher_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/tomasstrnad1997/mines_game_server_launcher"
	protocol "github.com/tomasstrnad1997/mines_protocol"
)

func TestGameLaunchViaTCP(t *testing.T){
	
	nServers := 5
	// Start server
	launcher, err := gamelauncher.CreateGameLauncher("mines.strnadt.cz", 42070)
	if err != nil {
		t.Logf("Launcher did not start: %v", err)
	}
	go launcher.ManageCommands()
	go launcher.Loop()

	// time.Sleep(300 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:42070")
	if err != nil {
		t.Fatalf("Cannot connect to game launcher: %v", err)
	}
	defer conn.Close()
	for i := range(nServers) {
		payload, err := protocol.EncodeSpawnServerRequest(fmt.Sprintf("Server %d", i))
		if err != nil {
			t.Fatalf("Failed to encode game start request: %v", err)
		}
		_, err = conn.Write(payload)
		if err != nil {
			t.Fatalf("Failed to send payload to spawn game server: %v", err)
		}
	}
	// time.Sleep(300*time.Millisecond)
	// Request game server info
	payload, err := protocol.EncodeGetGameServers()
	if err != nil {
		t.Fatalf("Failed encode GetGameServers: %v", err)
	}
	_, err = conn.Write(payload)
	if err != nil {
		t.Fatalf("Failed to send payload to get servers: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(1*time.Second))
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to recieve message %v", err)
	}
	data := buf[:n]
	servers, err := protocol.DecodeSendGameServers(data)

	if err != nil {
		t.Fatalf("Failed to recieve message %v", err)
	}
	// Go over server names and check if they are running
	for i := range(nServers) {
		name :=	fmt.Sprintf("Server %d", i)
		found := false
		for _, server := range servers {
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


