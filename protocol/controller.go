package protocol

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	maxReconnectAttempts = 100
)

type MessageHandler func([]byte) error

type Handler interface {
	HandleMessage(bytes []byte) error
}

type ConnectionController struct {
	server net.Conn
	messageHandlers map[MessageType]MessageHandler
	messageChannel chan []byte
	Connected bool
	host string
	port uint16
	AttemptReconnect bool
}

func (controller *ConnectionController) GetServerAddress() string {
	if !controller.Connected {
		return ""
	}
	addr := controller.server.RemoteAddr().(*net.TCPAddr)
	return fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)
}

func (controller *ConnectionController) StartWriter() {
	go func() {
		for {
			select {
			case message := <-controller.messageChannel:
				if !controller.Connected {
					fmt.Println("Attempted to write to not connected server")
				}
				// fmt.Printf("Sent: 0x%X %d\n", message[0], len(message))
				_, err := controller.server.Write(message)
				if err != nil {
					fmt.Println("Error writing to server:", err)
					return
				}
			}
		}
	}()
}

func (controller *ConnectionController) SendMessage(message []byte) error{
	select {
		case controller.messageChannel <- message:
		default:
			return fmt.Errorf("Failed to write to message channel")
	}
	return nil
}

func (controller *ConnectionController) SetConnection(conn net.Conn) error {
	if controller.Connected{
		return fmt.Errorf("Connector is already connected")
	}
	controller.server = conn
	controller.Connected = true
	return nil
}

func CreateConnectionController() *ConnectionController{
	messageHandlers := make(map[MessageType]MessageHandler)
	channel := make(chan []byte, 64)
	controller := &ConnectionController{messageHandlers: messageHandlers, Connected: false, messageChannel: channel}
	controller.StartWriter()
	return controller
}

func (controller *ConnectionController) HandleMessage(bytes []byte) error {
	msgType := MessageType(bytes[0])
	handlerFunc, exists := controller.messageHandlers[msgType]
	if !exists {
		return fmt.Errorf("No handler registered for message type: %d", msgType)
	}
	return handlerFunc(bytes)
}

func (controller *ConnectionController) connectLoop() error{
	attempts := 0
	for attempts < maxReconnectAttempts {
		err := controller.connect()
		if err == nil {
			fmt.Println("Connected successfully.")
			return nil
		}
		time.Sleep(time.Second * time.Duration(2))
		attempts++
	}
	return fmt.Errorf("Failed to connect after max attempts")
}

func (controller *ConnectionController) connect() error{
	if controller.Connected {
		return fmt.Errorf("Connector already connected")
	}
	server, err := connectUsingTcp(controller.host, controller.port)
	if err != nil {
		return err
	}
	controller.Connected = true
	controller.server = server
	return nil
}

func (controller *ConnectionController) Connect(host string, port uint16) error{
	controller.host = host
	controller.port = port
	return controller.connectLoop()
}

func (controller *ConnectionController) RegisterHandler(msgType MessageType, handlerFunc MessageHandler) {
	controller.messageHandlers[msgType] = handlerFunc
}

func connectUsingTcp(host string, port uint16) (*net.TCPConn, error){
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (controller *ConnectionController) readLoop() error {
	reader := bufio.NewReader(controller.server)
	for {
		header := make([]byte, HeaderLength)
		bytesRead, err := reader.Read(header)
		if err != nil {
			return err
		}
		if bytesRead != HeaderLength{
			return fmt.Errorf("Invalid message length read\n")
		}
		messageLenght := int(binary.BigEndian.Uint32(header[2:HeaderLength]))
		message := make([]byte, messageLenght+HeaderLength)
		copy(message[0:HeaderLength], header)
		_, err = io.ReadFull(reader, message[HeaderLength:])
		if err != nil {
			return err
		}
		// fmt.Printf("Recieved: 0x%X %d\n", message[0], len(message))
		if err = controller.HandleMessage(message); err != nil {
			return err
		}
	}
}

func (controller *ConnectionController) DeleteHandler(msgType MessageType) {
	delete(controller.messageHandlers, msgType)
}

func (controller *ConnectionController) ReadServerResponse() error{
	for {
		err := controller.readLoop()
		if err != nil {
			fmt.Println("Connection lost:", err)
			controller.Connected = false
			controller.server.Close()
			if controller.AttemptReconnect {
				fmt.Println("Attempting to reconnect...")
				controller.connectLoop()
			}else{
				return err
			}
		}

	}
	
}

