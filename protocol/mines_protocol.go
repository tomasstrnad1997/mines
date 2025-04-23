package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/tomasstrnad1997/mines/mines"
)


type MessageType byte
const (
    MoveCommand MessageType = 0x01
    TextMessage = 0x02
    Board = 0x03
    StartGame = 0x04
    CellUpdate = 0x05
    RequestReload = 0x06
    GameEnd = 0x07

	SpawnServerRequest = 0xA0
	SendGameServers = 0xA1
	GetGameServers = 0xA2
	PlayerSpawnServerRequest = 0xA3
	ServerSpawned = 0xA4
)


// Custom flags of special second byte
const (
	HasIdFlag byte = 0x01
)

type GameEndType byte
const (
    Win GameEndType = 0x01
    Loss = 0x02
    Aborted = 0x03
)

const (
    HeaderLength = 6
    CellByteLength = 9
    UpdateCellByteLength = 9
)

type GameServerInfo struct {
	Name string
	Host string //IP for clients to connect to
	Port uint16
	PlayerCount int
}



func checkAndDecodeLength(data []byte, message MessageType) (int, error){
    if len(data) < HeaderLength {
        return 0, fmt.Errorf("Data too short to decode")
    }
    if MessageType(data[0]) != message {
		return 0, fmt.Errorf("Invalid message type for command E:%d R:%d", message, data[0])
    }
    payloadLength := int(binary.BigEndian.Uint32(data[2:6]))
    if payloadLength != len(data) - HeaderLength {
        return payloadLength, fmt.Errorf("Payload size is invalid") // TODO: make a custom error 
    }
    return payloadLength, nil
}

func GetRequestId(data []byte, requestId *uint32) (error){
	if requestId == nil {
		return fmt.Errorf("RequestId pointer is nil")
	}
	if (data[1] & HasIdFlag) == 0{
		return fmt.Errorf("HasIdFlag not set so packet does not contain requestId")
	}
	if len(data) < HeaderLength + 4 {
		return fmt.Errorf("Data too short to retrieve requestId")
	}
	*requestId = binary.BigEndian.Uint32(data[HeaderLength:HeaderLength+4])
	return nil
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
    err := binary.Write(buf, binary.BigEndian, uint32(length))
    if err != nil {
        return fmt.Errorf("Failed to write length (%d)", length)
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

func EncodeServerSpawned(info *GameServerInfo, requestId uint32) ([]byte, error){
    var buf bytes.Buffer
    buf.WriteByte(byte(ServerSpawned))
    buf.WriteByte(byte(0x00))
	encoded, err := EncodeGameServer(info)
	if err != nil {
		return nil, err
	}

	writeLength(&buf, len(encoded)+4)
	err = binary.Write(&buf, binary.BigEndian, requestId)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(encoded)
	if err != nil {
		return nil, err
	}
    return buf.Bytes(), nil
}

func DecodeServerSpawned(data []byte) (*GameServerInfo, uint32, error){
    _, err := checkAndDecodeLength(data, ServerSpawned)
	if err != nil {
		return nil, 0, err
	}
	payload := data[HeaderLength:]
	requestId := binary.BigEndian.Uint32(payload[:4])
	buf := bytes.NewReader(payload[4:])
	server, err := DecodeGameServer(buf)
	if err != nil {
		return nil, 0, err
	}
	return server, requestId, nil
}

func EncodeGetGameServers(requestId *uint32) ([]byte, error) {
    var buf bytes.Buffer
    buf.WriteByte(byte(GetGameServers))
	var flags byte = 0x00
	payloadLength := 0
	if requestId != nil{
		payloadLength += 4
		flags |= HasIdFlag
	}
    buf.WriteByte(byte(flags))
	err := writeLength(&buf, payloadLength)
	if err != nil {
		return nil, err
	}

	if requestId != nil {
		if err := binary.Write(&buf, binary.BigEndian, *requestId); err != nil {
			return nil, err
		}
	}
    return buf.Bytes(), nil
}

func DecodeGetGameServers(data []byte, requestId *uint32) (error){
    _, err := checkAndDecodeLength(data, GetGameServers)
	if err != nil {
		return err
	}
	if requestId != nil {
		err = GetRequestId(data, requestId)
		if err != nil {
			return err
		}
	}
	return nil 
}

func DecodeSendGameServers(data []byte, requestId *uint32) ([]*GameServerInfo, error){
    _, err := checkAndDecodeLength(data, SendGameServers)
	if err != nil {
		return nil, err
	}
	offset := HeaderLength
	if requestId != nil {
		err = GetRequestId(data, requestId)
		if err != nil {
			return nil, err
		}
		offset += 4
	}
	buf := bytes.NewReader(data[offset:])
	servers := make([]*GameServerInfo, 0)
	for buf.Len() > 0 {
		server, err := DecodeGameServer(buf)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

func EncodeSendGameServers(servers []*GameServerInfo, requestId *uint32) ([]byte, error){
    var buf bytes.Buffer
    buf.WriteByte(byte(SendGameServers))
	var flags byte = 0x00
	payloadLength := 0
	if requestId != nil{
		payloadLength += 4
		flags |= HasIdFlag
	}
    buf.WriteByte(byte(flags))
	payloads := make([][]byte, len(servers))
	for i, server := range servers {
		encoded, err := EncodeGameServer(server)
		if err != nil {
			return nil, err
		}
		payloads[i] = encoded
		payloadLength += len(encoded)
	}

	writeLength(&buf, payloadLength)
	if requestId != nil {
		if err := binary.Write(&buf, binary.BigEndian, *requestId); err != nil {
			return nil, err
		}
	}

	for _, payload := range payloads {
		_, err := buf.Write(payload)
		if err != nil {
			return nil, err
		}
	}
    return buf.Bytes(), nil
}

func EncodeGameServer(server *GameServerInfo) ([]byte, error){
	// encoded structure |NameLength - int|name - string|HostLength - int|host - string|port - uint16|PlayerCount - int|
	// Total lengt = 4+NameLength+4+HostLength+2+4 = 14 + NameLengt + HostLength 
    var buf bytes.Buffer
	err := writeStringWithLength(&buf, server.Name)
    if err != nil {
        return nil, err
    }
	err = writeStringWithLength(&buf, server.Host)
    if err != nil {
        return nil, err
    }
	err = binary.Write(&buf, binary.BigEndian, server.Port)
    if err != nil {
        return nil, err
    }
	err = binary.Write(&buf, binary.BigEndian, int32(server.PlayerCount))
    if err != nil {
        return nil, err
    }
	return buf.Bytes(), nil
}

func DecodeGameServer(buf io.Reader) (*GameServerInfo, error) {
	name, err := readStringWithLength(buf)
	if err != nil {
		return nil, err
	}

	host, err := readStringWithLength(buf)
	if err != nil {
		return nil, err
	}

	var port uint16
	if err := binary.Read(buf, binary.BigEndian, &port); err != nil {
		return nil, err
	}

	var playerCount int32
	if err := binary.Read(buf, binary.BigEndian, &playerCount); err != nil {
		return nil, err
	}

	return &GameServerInfo{
		Name:        name,
		Host:        host,
		Port:        port,
		PlayerCount: int(playerCount),
	}, nil
}

func writeStringWithLength(buf *bytes.Buffer, str string) (error){
    err := writeLength(buf, len(str))
    if err != nil {
        return err
    }
	_, err = buf.WriteString(str)
	return err
}

func readStringWithLength(r io.Reader) (string, error) {
	var length int32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return "", err
	}

	strBytes := make([]byte, length)
	if _, err := io.ReadFull(r, strBytes); err != nil {
		return "", err
	}

	return string(strBytes), nil
}

func EncodeSpawnServerRequest(name string, requestId uint32) ([]byte, error){
    var buf bytes.Buffer
    buf.WriteByte(byte(SpawnServerRequest))
    buf.WriteByte(byte(0x00))
    err := writeLength(&buf, len(name)+4)
	err = binary.Write(&buf, binary.BigEndian, requestId)
    if err != nil {
        return nil, err
    }
	_, err = buf.WriteString(name)
	if err != nil {
		return nil, err
	}
    return buf.Bytes(), nil
}

func EncodePlayerSpawnServerRequest(name string) ([]byte, error){
    var buf bytes.Buffer
    buf.WriteByte(byte(PlayerSpawnServerRequest))
    buf.WriteByte(byte(0x00))
    err := writeLength(&buf, len(name))
    if err != nil {
        return nil, err
    }
	_, err = buf.WriteString(name)
	if err != nil {
		return nil, err
	}
    return buf.Bytes(), nil
}

func DecodeSpawnServerRequest(data []byte) (string, uint32, error){
    _, err := checkAndDecodeLength(data, SpawnServerRequest)
    if err != nil {
        return "", 0, err
    }
	payload := data[HeaderLength:]
	requestId := binary.BigEndian.Uint32(payload[:4])
	serverName := string(payload[4:])
	return serverName, requestId, nil
}

func DecodePlayerSpawnServerRequest(data []byte) (string, error){
    _, err := checkAndDecodeLength(data, PlayerSpawnServerRequest)
    if err != nil {
        return "", err
    }
	payload := data[HeaderLength:]
	return string(payload), nil
}

func DecodeGameEnd(data []byte) (GameEndType, error){
    _, err := checkAndDecodeLength(data, GameEnd)
    if err != nil {
        return 0, err
    }
    return GameEndType(data[HeaderLength]), nil
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
    payload := data[HeaderLength:]
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
    payload := data[HeaderLength:]
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
    for y := range board.Height{
        for x := range board.Width{
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

func DecodeBoard(data []byte) (*mines.Board, error){
    _, err := checkAndDecodeLength(data, Board)
    if err != nil {
        return nil, err
    }
    payload := data[HeaderLength:]
    
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
    if payloadLength + HeaderLength != buf.Len(){
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
    payload := data[HeaderLength:]
    if payloadLength % UpdateCellByteLength != 0 {
        return nil, fmt.Errorf("update cells payload length mismatch %d", payloadLength)
    }
    cells := make([]mines.UpdatedCell, payloadLength / UpdateCellByteLength)
    for i:=range payloadLength/UpdateCellByteLength{
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
    payload := data[HeaderLength:]
    params := &mines.GameParams{Width: bytesToInt(payload[0:4]),
                                Height: bytesToInt(payload[4:8]),
                                Mines: bytesToInt(payload[8:12])}
    return params, nil
}

