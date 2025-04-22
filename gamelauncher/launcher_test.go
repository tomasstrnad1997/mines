package gamelauncher_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/tomasstrnad1997/mines/gamelauncher"
	"github.com/tomasstrnad1997/mines/protocol"
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
		payload, err := protocol.EncodeSpawnServerRequest(fmt.Sprintf("Server %d", i), uint32(i))
		if err != nil {
			t.Fatalf("Failed to encode game start request: %v", err)
		}
		_, err = conn.Write(payload)
		if err != nil {
			t.Fatalf("Failed to send payload to spawn game server: %v", err)
		}
	}
	time.Sleep(100*time.Millisecond)
	// Go over server names and check if they are running
	for i := range(nServers) {
		name :=	fmt.Sprintf("Server %d", i)
		found := false
		for _, server := range launcher.Game_servers{
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


