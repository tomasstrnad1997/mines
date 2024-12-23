package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/tomasstrnad1997/mines"
)


type MessageType byte
type MessageHandler func([]byte) error
const (
    MoveCommand MessageType = 0x01
)




var messageHandlers = make(map[MessageType]MessageHandler)

func HandleMessage(bytes []byte) error {

    msgType := MessageType(bytes[0])
	handler, exists := messageHandlers[msgType]
	if !exists {
		return fmt.Errorf("No handler registered for message type: %d", msgType)
	}
	return handler(bytes)
}

func RegisterHandler(msgType MessageType, handler MessageHandler) {
	messageHandlers[msgType] = handler
}


func intToBytes(i int) []byte{
    buf := make([]byte, 4)
    binary.BigEndian.PutUint32(buf, uint32(i))
    return buf
}
func bytesToInt(bytes []byte) int{
    return int(binary.BigEndian.Uint32(bytes))
}

func EncodeMove(move mines.Move)([]byte, error){
    var buf bytes.Buffer
    buf.WriteByte(byte(MoveCommand))
    // Reserved byte for future use
    buf.WriteByte(byte(0x00))
    payload := make([]byte, 9)
    payload[0] = byte(move.Type);
    copy(payload[1:5], intToBytes(move.X))
    copy(payload[5:9], intToBytes(move.Y))
    payloadLength := len(payload)
    err := binary.Write(&buf, binary.BigEndian, uint16(payloadLength))
    if err != nil {
        return nil, fmt.Errorf("Failed to write payload (length: %d)", payloadLength)
    }
    buf.Write(payload)
    return buf.Bytes(), nil

}

func checkAndDecodeLength(data []byte, message MessageType) (int, error){
    if len(data) < 4 {
        return 0, fmt.Errorf("Data too short to decode")
    }
    if MessageType(data[0]) != message {
        return 0, fmt.Errorf("Invalid message type for command")
    }
    payloadLength := int(binary.BigEndian.Uint16(data[2:4]))
    if payloadLength != len(data) - 4 {
        return payloadLength, fmt.Errorf("Payload size is invalid") // TODO: make a custom error 
    }
    return payloadLength, nil
}


func DecodeMove(data []byte) (move* mines.Move, err error){
    _, err = checkAndDecodeLength(data, MoveCommand)
    if err != nil {
        return nil, err
    }
    move = &mines.Move{}
    payload := data[4:]
    move.Type = mines.MoveType(payload[0])
    move.X = bytesToInt(payload[1:5])
    move.Y = bytesToInt(payload[5:9])
    return move, nil
}

func main() {
    RegisterHandler(MoveCommand, func(bytes []byte) error {
        move, err := DecodeMove(bytes)
        if err != nil{
            return err
        }
        println((*move).String())
        return nil

    })
    move := mines.Move{X:5, Y:5, Type:0x02} 
    encoded, err := EncodeMove(move)
    if err != nil{
        println(err.Error())
    }
    mines.CreateBoard(10, 10, 10)
    HandleMessage(encoded)

}
