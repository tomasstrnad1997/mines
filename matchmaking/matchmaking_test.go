package matchmaking_test

import (
	"bufio"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/tomasstrnad1997/mines/db"
	"github.com/tomasstrnad1997/mines/gamelauncher"
	"github.com/tomasstrnad1997/mines/matchmaking"
	"github.com/tomasstrnad1997/mines/protocol"
)

type MMserverOptions struct {
	port   uint16
	dbPath string
	tempDB bool
}

func setupMMserverAndLauncher(t *testing.T, launcherPort uint16, mmOpts MMserverOptions) (*matchmaking.MatchmakingServer, *gamelauncher.GameLauncher) {
	t.Helper()
	mmServer := setupMMserver(t, mmOpts)
	// Start game launcher and connect to it from mm server
	launcher, err := gamelauncher.CreateGameLauncher("localhost", launcherPort)
	if err != nil {
		log.Fatalf("Failed to create GameLauncher: %v", err)
	}
	go launcher.Loop()
	mmServer.ConnectToLauncher("localhost", launcherPort, true)
	return mmServer, launcher
}

func copyDB(t *testing.T, originalDbPath string) string {
	t.Helper()
	tempFile := createTempDBFile(t)
	defer tempFile.Close()
	src, err := os.Open(originalDbPath)
	if err != nil {
		t.Fatalf("Failed to open orignal database file: %v", err)
	}
	defer src.Close()

	if _, err = io.Copy(tempFile, src); err != nil {
		t.Fatalf("Failed to copy db to a temp file: %v", err)
	}

	return tempFile.Name()
}

func createTempDBFile(t *testing.T) *os.File {
	t.Helper()
	tempFile, err := os.CreateTemp("", "*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file for db copy: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			fmt.Printf("failed to delete DB file %v\n", err)
		}
	})
	return tempFile
}

func createEmptyTempDB(t *testing.T) string {
	t.Helper()
	tempFile := createTempDBFile(t)
	tempFile.Close()
	database, err := sql.Open("sqlite3", tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to open db file: %v", err)
	}

	if err = db.InitializeTables(database); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}
	database.Close()
	return tempFile.Name()

}

func setupMMserver(t *testing.T, opts MMserverOptions) *matchmaking.MatchmakingServer {
	t.Helper()
	if opts.tempDB && len(opts.dbPath) > 0 {
		t.Fatalf("Cant have tempDB and path set once")
	}
	if opts.tempDB {
		tempName := createEmptyTempDB(t)
		opts.dbPath = tempName
	}
	dbFilename := copyDB(t, opts.dbPath)
	os.Setenv("DB_PATH", dbFilename)
	mmServer, err := matchmaking.CreateMatchMakingServer(opts.port)

	if err != nil {
		log.Fatalf("Failed to create MM server: %v", err)
	}
	go mmServer.Run()
	return mmServer
}

func waitForResponse(conn net.Conn, t *testing.T) []byte {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	reader := bufio.NewReader(conn)
	header := make([]byte, protocol.HeaderLength)
	bytesRead, err := reader.Read(header)
	if err != nil {
		t.Fatalf("Lost connection to server: %v", err)
	}
	if bytesRead != protocol.HeaderLength {
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

func TestRegisterPlayer(t *testing.T) {
	mmPort := uint16(42099)
	mmOpts := MMserverOptions{port: mmPort, tempDB: true}
	mmServer := setupMMserver(t, mmOpts)
	player := protocol.AuthPlayerParams{Name: "John", Password: "password+123"}
	encoded, err := protocol.EncodeRegisterPlayerRequest(player)
	if err != nil {
		t.Fatalf("Failed to encode register player data: %v", err)
	}
	// Connect to matchmaking server as a player
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", mmPort))
	if err != nil {
		t.Fatalf("Cannot connect to game launcher: %v", err)
	}
	_, err = conn.Write(encoded)
	if err != nil {
		t.Fatalf("Failed to write to server: %v", err)
	}
	// Sleep since pw hash and db store takes some time
	time.Sleep(time.Millisecond * 100)
	if _, err := mmServer.PlayerService.Login(player.Name, player.Password); err != nil {
		t.Fatalf("Failed to retrieve player by name: %v", err)
	}
}

func TestGetServerList(t *testing.T) {
	mmPort := uint16(42075)
	launcherPort := uint16(42076)
	nServers := 5
	mmOpts := MMserverOptions{port: mmPort, tempDB: true}
	_, launcher := setupMMserverAndLauncher(t, launcherPort, mmOpts)
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
	for i := range nServers {
		name := fmt.Sprintf("Server %d", i)
		found := false
		for _, server := range infos {
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

func TestGameServerSpawn(t *testing.T) {
	mmPort := uint16(42071)
	launcherPort := uint16(42070)
	mmOpts := MMserverOptions{port: mmPort, tempDB: true}
	setupMMserverAndLauncher(t, launcherPort, mmOpts)

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
