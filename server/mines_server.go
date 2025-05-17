package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/tomasstrnad1997/mines/mines"
	"github.com/tomasstrnad1997/mines/players"
	"github.com/tomasstrnad1997/mines/protocol"
)

type Player struct {
	localID        int
	controller     *protocol.ConnectionController
	info           *players.PlayerInfo
	authenticated  bool
	authResponseCh chan bool
}

type MessageHandler func(data []byte, source int) error

type command struct {
	message []byte
	player  *Player
}
type Server struct {
	id             int
	Name           string
	server         net.Listener
	game           *mines.Game
	gameRunning    bool
	handlers       map[protocol.MessageType]MessageHandler
	messageChannel chan command
	Port           uint16
	clients        map[int]bool
	players        map[int]*Player
	clientsMux     sync.Mutex
	moveMux        sync.Mutex
	requiresAuth   bool
	authSecret     []byte
}

func (server *Server) GetNumberOfPlayers() int {
	count := 0
	for _, player := range server.players {
		if player.controller.Connected {
			count++
		}
	}
	return count
}

func (server *Server) GetServerInfo() *protocol.GameServerInfo {
	return &protocol.GameServerInfo{Name: server.Name, Host: "", Port: server.Port, PlayerCount: server.GetNumberOfPlayers()}

}

func (server *Server) StartGame(params mines.GameParams) error {
	game, err := mines.CreateGame(params)
	if err != nil {
		return err
	}
	server.game = game
	//server.broadcastTextMessage(fmt.Sprintf("Starting a new game...\nNumber of mines %d", params.Mines))

	println("Starting a new game")
	startMsg, err := protocol.EncodeGameStart(params)
	if err != nil {
		return err
	}
	server.broadcast(startMsg)
	server.gameRunning = true
	return nil
}

func (server *Server) broadcastTextMessage(message string) {
	encoded, err := protocol.EncodeTextMessage(message)
	if err != nil {
		println("Failed to create message")
		return
	}
	server.broadcast(encoded)
}

func (server *Server) broadcast(data []byte) {
	for _, player := range server.players {
		if player.controller.Connected {
			player.controller.SendMessage(data)
		}
	}
}
func sendTextMessage(msg string, player *Player) {
	encoded, err := protocol.EncodeTextMessage(msg)
	if err != nil {
		println("Failed to create a message")
		return
	}
	sendMessage(encoded, player)
}

func sendMessage(data []byte, player *Player) {
	if player.controller.Connected {
		player.controller.SendMessage(data)
	}
}

func (server *Server) sendInitialMessages(player *Player) error {
	startMsg, err := protocol.EncodeGameStart(server.game.Params)
	if err != nil {
		return err
	}
	sendMessage(startMsg, player)
	cellUpdates, err := server.game.GetChangedCellUpdates()
	if err != nil {
		return err
	}
	updateMsg, err := protocol.EncodeCellUpdates(cellUpdates)
	if err != nil {
		return err
	}
	sendMessage(updateMsg, player)

	return nil
}

func (player *Player) RegisterAuthHandlers(server *Server) {
	player.controller.RegisterHandler(protocol.AuthWithMMToken, func(bytes []byte) error {
		token, err := protocol.DecodeAuthWithMMToken(bytes)
		if err != nil {
			return err
		}
		success, err := players.ValidateAuthToken(token, server.authSecret)
		player.authResponseCh <- success
		if err != nil {
			return err
		}
		player.authenticated = success
		return nil
	})
}

func RegisterHandlers(player *Player, server *Server) {
	player.controller.RegisterHandler(protocol.StartGame, func(bytes []byte) error {
		params, err := protocol.DecodeGameStart(bytes)
		if err != nil {
			return err
		}
		if server.gameRunning {
			msg, err := protocol.EncodeGameEnd(protocol.Aborted)
			if err != nil {
				return err
			}
			server.broadcast(msg)
		}
		//server.broadcastTextMessage(fmt.Sprintf("Player %d requested new game", player.id))
		return server.StartGame(*params)
	})
	player.controller.RegisterHandler(protocol.MoveCommand, func(bytes []byte) error {
		if !server.gameRunning {
			//sendTextMessage("Game not running. Cant make moves.", player)
			return nil
		}
		move, err := protocol.DecodeMove(bytes)
		if err != nil {
			return err
		}
		move.PlayerId = player.localID
		server.moveMux.Lock()
		moveResult, gamemodeInfo, err := server.game.MakeMove(*move)
		server.moveMux.Unlock()
		if err != nil {
			return err
		}
		if len(moveResult.UpdatedCells) > 0 {
			cells, err := server.game.CreateCellUpdates(moveResult.UpdatedCells)
			if err != nil {
				return err
			}
			encoded, err := protocol.EncodeCellUpdates(cells)
			if err != nil {
				return err
			}
			server.broadcast(encoded)
			if gamemodeInfo != nil {
				encoded, err = protocol.EncodeGamemodeInfo(gamemodeInfo)
				if err != nil {
					return err
				}
				server.broadcast(encoded)
			}
		}
		var endMsg []byte
		switch moveResult.Result {
		case mines.MineBlown:
			endMsg, err = protocol.EncodeGameEnd(protocol.Loss)
		case mines.GameWon:
			endMsg, err = protocol.EncodeGameEnd(protocol.Win)
		default:
			endMsg, err = nil, nil
		}
		if err != nil {
			return err
		}
		if endMsg != nil {
			server.broadcast(endMsg)
			server.gameRunning = false
		}
		return nil
	})
}

func createServer(id int, name string, port uint16) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Println("Failed to start server:", err.Error())
		return nil, err
	}
	handlers := make(map[protocol.MessageType]MessageHandler)
	messageChannel := make(chan command)
	serverPort := listener.Addr().(*net.TCPAddr).Port
	clients := make(map[int]bool)
	players := make(map[int]*Player)
	server := &Server{
		id:             id,
		Name:           name,
		server:         listener,
		game:           nil,
		gameRunning:    false,
		handlers:       handlers,
		messageChannel: messageChannel,
		Port:           uint16(serverPort),
		clients:        clients,
		players:        players,
		authSecret:     []byte(os.Getenv("AUTH_SECRET")),
	}
	return server, nil
}

func playerAcceptLoop(server *Server) {
	defer server.server.Close()
	id := 1
	for {
		conn, err := server.server.Accept()
		if err != nil {
			println(err)
			return
		}
		go server.handleNewConnection(conn, id)
		id++
	}
}

func (server *Server) handleNewConnection(conn net.Conn, localId int) {
	controller := protocol.CreateConnectionController()
	if err := controller.SetConnection(conn); err != nil {
		println(err)
		return
	}
	player := &Player{
		localID:        localId,
		controller:     controller,
		authResponseCh: make(chan bool),
	}
	server.players[player.localID] = player
	if server.requiresAuth {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		player.RegisterAuthHandlers(server)
		go controller.ReadServerResponse()
		select {
		case <-ctx.Done():
			// TODO: close connection
			fmt.Printf("Timeout waiting for auth response")
			return
		case authSuccess := <-player.authResponseCh:
			fmt.Printf("Auth response recieved")
			if !authSuccess {
				fmt.Printf("Player auth failed")
				// TODO: close connection
				return
			}
			fmt.Printf("Player auth was successful")

		}
		RegisterHandlers(player, server)
	} else {
		player.authenticated = true
		RegisterHandlers(player, server)
		go controller.ReadServerResponse()
	}
	if server.gameRunning {
		server.sendInitialMessages(player)
	}
}

func SpawnServer(id int, name string, port uint16) (*Server, error) {
	server, err := createServer(id, name, port)
	if err != nil {
		return nil, err
	}
	go playerAcceptLoop(server)
	return server, nil
}
