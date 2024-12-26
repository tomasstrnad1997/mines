package protocol

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
    TextMessage = 0x02
    Board = 0x03
    StartGame = 0x04
    CellUpdate = 0x05
    RequestReload = 0x06
    GameEnd = 0x07
)

type GameEndType byte
const (
    Win GameEndType = 0x01
    Loss = 0x02
    Aborted = 0x03
)




var messageHandlers = make(map[MessageType]MessageHandler)

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

func HandleMessage(bytes []byte) error {

    msgType := MessageType(bytes[0])
	handler, exists := messageHandlers[msgType]
	if !exists {
		return fmt.Errorf("No handler registered for message type: %d", msgType)
	}
	return handler(bytes)
}

//TODO: Do this differently on client or server side so it can access player data etc
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

func writeLength(buf *bytes.Buffer, length int) error {
    err := binary.Write(buf, binary.BigEndian, uint16(length))
    if err != nil {
        return fmt.Errorf("Failed to write payload length (%d)", length)
    }
    return nil
}

func EncodeGameEnd(endType GameEndType) ([]byte, error){
    var buf bytes.Buffer
    buf.WriteByte(byte(GameEnd))
    buf.WriteByte(byte(0x00))
    err := writeLength(&buf, 1)
    if err != nil {
        return nil, err
    }
    buf.WriteByte(byte(endType))
    return buf.Bytes(), nil
}

func DecodeGameEnd(data []byte) (GameEndType, error){
    _, err := checkAndDecodeLength(data, GameEnd)
    if err != nil {
        return 0, err
    }
    return GameEndType(data[4]), nil

}

func EncodeTextMessage(message string) ([]byte, error){
    var buf bytes.Buffer
    buf.WriteByte(byte(TextMessage))
    buf.WriteByte(byte(0x00))
    payload := []byte(message)
    err := writeLength(&buf, len(payload))
    if err != nil {
        return nil, err
    }
    buf.Write(payload)
    return buf.Bytes(), nil
}

func DecodeTextMessage(data []byte) (string, error){
    _, err := checkAndDecodeLength(data, TextMessage)
    if err != nil {
        return "", err
    }
    payload := data[4:]
    return string(payload), nil
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
    err := writeLength(&buf, len(payload))
    if err != nil {
        return nil, err
    }
    buf.Write(payload)
    return buf.Bytes(), nil

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

const (
    MineFlag    byte = 0b0001
    RevealedFlag byte = 0b0010
    FlaggedFlag  byte = 0b0100
)

func encodeCellFlags(cell *mines.Cell) byte {
    var flags byte = 0x00
    if cell.Mine {
        flags |= MineFlag
    }
    if cell.Revealed {
        flags |= RevealedFlag
    }
    if cell.Flagged {
        flags |= FlaggedFlag
    }
    return flags
}

func encodeCell(cell *mines.Cell) []byte {
    encoded := make([]byte, 9)
    copy(encoded[0:4], intToBytes(cell.X))
    copy(encoded[4:8], intToBytes(cell.Y))
    encoded[8] = encodeCellFlags(cell)
    return encoded
}

func decodeCellFlags(flags byte, cell *mines.Cell){
    cell.Mine = (flags & MineFlag) != 0
    cell.Revealed = (flags & RevealedFlag) != 0
    cell.Flagged = (flags & FlaggedFlag) != 0
}

func decodeCell(data []byte) (*mines.Cell, error){
    if len(data) != CellByteLength{
        return nil, fmt.Errorf("Invalid length to decode cell (%d)", len(data))
    }
    cell := mines.Cell{}
    cell.X = bytesToInt(data[0:4])
    cell.Y = bytesToInt(data[4:8])
    flags := data[8]
    decodeCellFlags(flags, &cell)
    return &cell, nil
}

func EncodeBoard(board *mines.Board) ([]byte, error){
    var buf bytes.Buffer
    buf.WriteByte(byte(Board))
    buf.WriteByte(byte(0x00))
    var boardBuf bytes.Buffer
    boardBuf.Write(intToBytes(board.Height))
    boardBuf.Write(intToBytes(board.Width))
    for y := 0; y < board.Height; y++{
        for x := 0; x < board.Width; x++{
            boardBuf.Write(encodeCell(board.Cells[x][y]))
        }
    }
    err := writeLength(&buf, boardBuf.Len())
    if err != nil{
        return nil, err
    }
    buf.Write(boardBuf.Bytes())
    return buf.Bytes(), nil

}

const (
    CellByteLength = 9
    UpdateCellByteLength = 9
)

func DecodeBoard(data []byte) (*mines.Board, error){
    _, err := checkAndDecodeLength(data, Board)
    if err != nil {
        return nil, err
    }
    payload := data[4:]
    
    if len(payload) < 8 {
        return nil, fmt.Errorf("payload too short to contain board dimensions")
    }

    board := &mines.Board{}
    board.Height = bytesToInt(payload[0:4])
    board.Width = bytesToInt(payload[4:8])
    grid := make([][]*mines.Cell, board.Width)
    for i := range grid {
        grid[i] = make([]*mines.Cell, board.Height)
    }
    board.Cells = grid
    cells := payload[8:]
    if len(cells) % CellByteLength != 0{
        return nil, fmt.Errorf("Cells payload length mismatch")
    }
    if len(cells) / CellByteLength != board.Height * board.Width{
        return nil, fmt.Errorf("Number of cells doesnt match board size")
    }
    for i := 0; i < len(cells); i+= CellByteLength{
        cell, err := decodeCell(cells[i: i+CellByteLength]) 
        if err != nil {
            return nil, err
        }
        if cell.X < 0 || cell.X >= board.Width || cell.Y < 0 || cell.Y >= board.Height {
            return nil, fmt.Errorf("Cell position out of bounds: (%d, %d)", cell.X, cell.Y)
        }
        if board.Cells[cell.X][cell.Y] != nil{
            return nil, fmt.Errorf("Duplicate entry of a cell")
        }
        board.Cells[cell.X][cell.Y] = cell
    }

    return board, nil
}

func encodeCellUpdate(cell mines.UpdatedCell) []byte {
    data := make([]byte, UpdateCellByteLength)
    copy(data[0:4], intToBytes(cell.X))
    copy(data[4:8], intToBytes(cell.Y))
    data[8] = cell.Value
    return data

}

func EncodeCellUpdates(cells []mines.UpdatedCell) ([]byte, error) {
    var buf bytes.Buffer
    buf.WriteByte(byte(CellUpdate))
    buf.WriteByte(byte(0x00))
    payloadLength := len(cells) * UpdateCellByteLength
    err := writeLength(&buf, payloadLength)
    if err != nil {
        return nil, err
    }
    for _, cell := range cells {
        buf.Write(encodeCellUpdate(cell))
    }
    if payloadLength + 4 != buf.Len(){
        return nil, fmt.Errorf("Incorrect payload length while encoding cell updates")
    }
    return buf.Bytes(), nil
}
 
func decodeCellUpdate(data []byte) (*mines.UpdatedCell, error){
    if len(data) != UpdateCellByteLength {
        return nil, fmt.Errorf("incorrect byte length to decode cell update (%d)", len(data))
    }
    cell := &mines.UpdatedCell{
        X: bytesToInt(data[0:4]),
        Y: bytesToInt(data[4:8]),
        Value: data[8]}
    return cell, nil
    
}

func DecodeCellUpdates(data []byte) ([]mines.UpdatedCell, error) {

    payloadLength, err := checkAndDecodeLength(data, CellUpdate)
    if err != nil {
        return nil, err
    }
    payload := data[4:]
    if payloadLength % UpdateCellByteLength != 0 {
        return nil, fmt.Errorf("update cells payload length mismatch")
    }
    cells := make([]mines.UpdatedCell, payloadLength / UpdateCellByteLength)
    for i:=0; i<payloadLength/UpdateCellByteLength; i++{
        cell, err := decodeCellUpdate(payload[i*UpdateCellByteLength: (i+1)*UpdateCellByteLength])
        if err != nil {
            return nil, err
        }
        cells[i] = *cell
    }
    return cells, nil
}

func EncodeGameStart(params mines.GameParams) ([]byte, error) {
    payloadLength := 3*4
    var buf bytes.Buffer
    buf.WriteByte(byte(StartGame))
    buf.WriteByte(byte(0x00))
    err := writeLength(&buf, payloadLength)
    if err != nil {
        return nil, err
    }
    payload := make([]byte, payloadLength)
    copy(payload[0:4], intToBytes(params.Width))
    copy(payload[4:8], intToBytes(params.Height))
    copy(payload[8:12], intToBytes(params.Mines))
    buf.Write(payload)
    return buf.Bytes(), nil
    
}

func DecodeGameStart(data []byte) (*mines.GameParams, error){
    payloadLength, err := checkAndDecodeLength(data, StartGame)
    if err != nil {
        return nil, err
    }
    if payloadLength != 3*4 {
        return nil, fmt.Errorf("decode game starte payload incorrect length (%d)", payloadLength)
    }
    payload := data[4:]
    params := &mines.GameParams{Width: bytesToInt(payload[0:4]),
                                Height: bytesToInt(payload[4:8]),
                                Mines: bytesToInt(payload[8:12])}
    return params, nil
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
