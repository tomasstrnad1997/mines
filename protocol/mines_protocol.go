package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/tomasstrnad1997/mines/mines"
	"github.com/tomasstrnad1997/mines/players"
)

type MessageType byte

const (
	MoveCommand   MessageType = 0x01
	TextMessage               = 0x02
	Board                     = 0x03
	StartGame                 = 0x04
	CellUpdate                = 0x05
	RequestReload             = 0x06
	GameEnd                   = 0x07
	GamemodeInfo              = 0x08

	SpawnServerRequest = 0xA0
	SendGameServers    = 0xA1
	GetGameServers     = 0xA2
	ServerSpawned      = 0xA3

	RegisterPlayerRequest  = 0xC0
	RegisterPlayerResponse = 0xC1
	AuthRequest            = 0xC2
	AuthResponseMessage    = 0xC3
	ConnectToGameRequest   = 0xC4
	ConnectToGameResponse  = 0xC5
	AuthWithMMToken        = 0xC6
)

// Custom flags of special second byte
const (
	HasIdFlag byte = 0x01
)

type GameEndType byte

const (
	Win     GameEndType = 0x01
	Loss                = 0x02
	Aborted             = 0x03
)

const (
	HeaderLength         = 6
	CellByteLength       = 9
	UpdateCellByteLength = 9
)

var (
	ErrInvalidPayloadSize = errors.New("invalid payload size")
)

type AuthResponse struct {
	Success bool
	Player  *players.PlayerInfo
}

type GameServerInfo struct {
	Name string
	// TODO: Remove host and port and add id of game
	Host        string
	Port        uint16
	PlayerCount int
}

type GameServerConnectInfo struct {
	Host string //IP for clients to connect to
	Port uint16
}

type AuthPlayerParams struct {
	Name     string
	Password string
}

type GameConnectionResponse struct {
	Success  bool
	Token    *players.AuthToken
	GameInfo *GameServerConnectInfo
}

func checkAndDecodeLength(data []byte, message MessageType) (int, error) {
	if len(data) < HeaderLength {
		return 0, fmt.Errorf("Data too short to decode")
	}
	if MessageType(data[0]) != message {
		return 0, fmt.Errorf("Invalid message type for command E:%d R:%d", message, data[0])
	}
	payloadLength := int(binary.BigEndian.Uint32(data[2:6]))
	if payloadLength != len(data)-HeaderLength {
		return payloadLength, fmt.Errorf("Payload size is invalid") // TODO: make a custom error
	}
	return payloadLength, nil
}

func GetRequestId(data []byte, requestId *uint32) error {
	if requestId == nil {
		return fmt.Errorf("RequestId pointer is nil")
	}
	if (data[1] & HasIdFlag) == 0 {
		return fmt.Errorf("HasIdFlag not set so packet does not contain requestId")
	}
	if len(data) < HeaderLength+4 {
		return fmt.Errorf("Data too short to retrieve requestId")
	}
	*requestId = binary.BigEndian.Uint32(data[HeaderLength : HeaderLength+4])
	return nil
}

func intToBytes(i int) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

func bytesToInt(bytes []byte) int {
	return int(binary.BigEndian.Uint32(bytes))
}

func writePayloadLength(buf *bytes.Buffer, length int) error {
	err := binary.Write(buf, binary.BigEndian, uint32(length))
	if err != nil {
		return fmt.Errorf("Failed to write length (%d)", length)
	}
	return nil
}

func EncodeAuthWithMMToken(token players.AuthToken) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(AuthWithMMToken))
	buf.WriteByte(byte(0x00))
	if err := writePayloadLength(&buf, players.AuthTokenLength); err != nil {
		return nil, err
	}
	encoded := encodeAuthToken(token)
	if _, err := buf.Write(encoded); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeAuthWithMMToken(data []byte) (players.AuthToken, error) {
	if _, err := checkAndDecodeLength(data, AuthWithMMToken); err != nil {
		return players.AuthToken{}, err
	}
	return decodeAuthToken(data[HeaderLength:])
}

func EncodeAuthRequest(params AuthPlayerParams) ([]byte, error) {
	return encodeAuthPlayerParamsMessage(params, AuthRequest)
}

func DecodeAuthRequest(data []byte) (*AuthPlayerParams, error) {
	return decodeAuthPlayerParams(data, AuthRequest)
}

func encodeAuthPlayerParamsMessage(params AuthPlayerParams, tp MessageType) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(tp))
	buf.WriteByte(byte(0x00))
	payload, err := encodePlayerParams(params)
	if err != nil {
		return nil, err
	}
	if err := writePayloadLength(&buf, len(payload)); err != nil {
		return nil, err
	}
	if _, err := buf.Write(payload); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func EncodeConnectToGameResponse(response GameConnectionResponse) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(ConnectToGameResponse))
	buf.WriteByte(byte(0x00))
	if !response.Success {
		if err := writePayloadLength(&buf, 1); err != nil {
			return nil, err
		}
		if err := buf.WriteByte(0); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	if response.Token == nil || response.GameInfo == nil {
		return nil, fmt.Errorf("Token and Game info can't be nill when succes == true")
	}

	gInfo, err := encodeGameServerConnectInfo(*response.GameInfo)
	if err != nil {
		return nil, err
	}
	token := encodeAuthToken(*response.Token)
	payloadLenth := 1 + len(gInfo) + len(token)
	if err := writePayloadLength(&buf, payloadLenth); err != nil {
		return nil, err
	}

	if err := buf.WriteByte(1); err != nil {
		return nil, err
	}

	if _, err := buf.Write(token); err != nil {
		return nil, err
	}

	if _, err := buf.Write(gInfo); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DecodeConnectToGameResponse(data []byte) (*GameConnectionResponse, error) {
	pLen, err := checkAndDecodeLength(data, ConnectToGameResponse)
	if err != nil {
		return nil, err
	}
	if pLen == 0 {
		return nil, ErrInvalidPayloadSize
	}
	payload := data[HeaderLength:]

	// Denied request
	if payload[0] != 1 {
		if pLen != 1 {
			return nil, ErrInvalidPayloadSize
		}
		return &GameConnectionResponse{Success: false}, nil
	}

	// Success + token + hostLen + host (atleast 1 char) + port
	if pLen < 1+players.AuthTokenLength+4+1+2 {
		return nil, ErrInvalidPayloadSize
	}

	token, err := decodeAuthToken(payload[1 : 1+players.AuthTokenLength])
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(payload[1+players.AuthTokenLength:])
	gsInfo, err := decodeGameServerConnectInfo(buf)
	if err != nil {
		return nil, err
	}

	return &GameConnectionResponse{
		Success:  true,
		Token:    &token,
		GameInfo: gsInfo,
	}, nil
}

func decodeAuthToken(data []byte) (players.AuthToken, error) {
	if len(data) != players.AuthTokenLength {
		return players.AuthToken{}, fmt.Errorf("invalid data size to decode auth token: %d", len(data))
	}
	var token players.AuthToken
	token.PlayerID = binary.BigEndian.Uint32(data[0:4])
	token.Expiry = int64(binary.BigEndian.Uint64(data[4:12]))
	copy(token.Nonce[:], data[12:28])
	copy(token.Signature[:], data[28:players.AuthTokenLength])
	return token, nil
}

func encodeAuthToken(token players.AuthToken) []byte {
	encoded := make([]byte, players.AuthTokenLength)
	binary.BigEndian.PutUint32(encoded[0:4], token.PlayerID)
	binary.BigEndian.PutUint64(encoded[4:12], uint64(token.Expiry))
	copy(encoded[12:28], token.Nonce[:])
	copy(encoded[28:players.AuthTokenLength], token.Signature[:])

	return encoded
}

func encodeGameServerConnectInfo(server GameServerConnectInfo) ([]byte, error) {
	// encoded structure |HostLength - int|host - string|port - uint16|
	// Total lengt = 4+HostLength+2 = 6 + HostLength
	var buf bytes.Buffer
	if err := writeStringWithLength(&buf, server.Host); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.BigEndian, server.Port); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeGameServerConnectInfo(buf io.Reader) (*GameServerConnectInfo, error) {
	host, err := readStringWithLength(buf)
	if err != nil {
		return nil, err
	}
	var port uint16
	if err := binary.Read(buf, binary.BigEndian, &port); err != nil {
		return nil, err
	}
	return &GameServerConnectInfo{Host: host, Port: port}, nil
}

func EncodeConnectToGameRequest(gameServerID uint32) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(ConnectToGameRequest))
	buf.WriteByte(byte(0x00))
	if err := writePayloadLength(&buf, 4); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.BigEndian, gameServerID); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeConnectToGameRequest(data []byte) (uint32, error) {
	length, err := checkAndDecodeLength(data, ConnectToGameRequest)
	if err != nil {
		return 0, err
	}
	if length != 4 {
		return 0, fmt.Errorf("ConnectToGameServerRequest payload lentth != 4 (got %d)", length)
	}
	return binary.BigEndian.Uint32(data[HeaderLength:]), nil
}

func EncodeAuthResponse(response AuthResponse) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(AuthResponseMessage))
	buf.WriteByte(byte(0x00))
	if !response.Success {
		if err := writePayloadLength(&buf, 1); err != nil {
			return nil, err
		}
		if err := buf.WriteByte(0); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	if response.Player == nil {
		return nil, fmt.Errorf("player cannot be nil when success is true")
	}
	// Success + playerID + nameLen + name
	payloadLength := 1 + 4 + 4 + len(response.Player.Name)
	if err := writePayloadLength(&buf, payloadLength); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.BigEndian, response.Player.ID); err != nil {
		return nil, err
	}
	if err := writeStringWithLength(&buf, response.Player.Name); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeAuthResponse(data []byte) (*AuthResponse, error) {
	pLen, err := checkAndDecodeLength(data, AuthResponseMessage)
	if err != nil {
		return nil, err
	}
	if pLen == 0 {
		return nil, ErrInvalidPayloadSize
	}
	payload := data[HeaderLength:]

	// Auth failed
	if payload[0] != 1 {
		if pLen != 1 {
			return nil, ErrInvalidPayloadSize
		}
		return &AuthResponse{Success: false}, nil
	}

	// Success + id + nameLen + name (atleast 1 char)
	if pLen < 1+4+4+1 {
		return nil, ErrInvalidPayloadSize
	}
	id := binary.BigEndian.Uint32(payload[1:5])
	nameLen := binary.BigEndian.Uint32(payload[5:9])
	name := string(payload[9 : 9+nameLen])
	return &AuthResponse{
		Success: true,
		Player: &players.PlayerInfo{
			ID:   id,
			Name: name,
		},
	}, nil

}

func EncodeRegisterPlayerResponse(success bool) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(RegisterPlayerResponse))
	buf.WriteByte(byte(0x00))
	if err := writePayloadLength(&buf, 1); err != nil {
		return nil, err
	}
	var b byte = 0
	if success {
		b = 1
	}
	if err := buf.WriteByte(b); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeRegisterPlayerResponse(data []byte) (bool, error) {
	_, err := checkAndDecodeLength(data, ServerSpawned)
	if err != nil {
		return false, err
	}
	success := data[HeaderLength] == 1
	return success, nil
}

func EncodeRegisterPlayerRequest(params AuthPlayerParams) ([]byte, error) {
	return encodeAuthPlayerParamsMessage(params, RegisterPlayerRequest)
}

func encodePlayerParams(args AuthPlayerParams) ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, int32(len(args.Name))); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(args.Name + args.Password); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeAuthPlayerParams(data []byte, tp MessageType) (*AuthPlayerParams, error) {
	_, err := checkAndDecodeLength(data, tp)
	if err != nil {
		return nil, err
	}
	payload := data[HeaderLength:]
	nameLen := bytesToInt(payload[0:4])
	passwordOffset := nameLen + 4
	name := string(payload[4:passwordOffset])
	password := string(payload[passwordOffset:])
	params := &AuthPlayerParams{Name: name, Password: password}
	return params, nil
}

func DecodeRegisterPlayerRequest(data []byte) (*AuthPlayerParams, error) {
	return decodeAuthPlayerParams(data, RegisterPlayerRequest)
}

func EncodeGameEnd(endType GameEndType) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(GameEnd))
	buf.WriteByte(byte(0x00))
	err := writePayloadLength(&buf, 1)
	if err != nil {
		return nil, err
	}
	buf.WriteByte(byte(endType))
	return buf.Bytes(), nil
}

func EncodeServerSpawned(info *GameServerInfo, requestId *uint32) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(ServerSpawned))
	var flag byte = 0x00
	offset := 0
	if requestId != nil {
		flag |= HasIdFlag
		offset += 4
	}
	encoded, err := EncodeGameServer(info)
	if err != nil {
		return nil, err
	}

	buf.WriteByte(byte(flag))
	writePayloadLength(&buf, len(encoded)+offset)
	if requestId != nil {
		if err = binary.Write(&buf, binary.BigEndian, requestId); err != nil {
			return nil, err
		}
	}
	_, err = buf.Write(encoded)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeServerSpawned(data []byte, requestId *uint32) (*GameServerInfo, error) {
	_, err := checkAndDecodeLength(data, ServerSpawned)
	if err != nil {
		return nil, err
	}
	offset := HeaderLength
	if requestId != nil {
		if err = GetRequestId(data, requestId); err != nil {
			return nil, err
		}
		offset += 4

	}
	payload := data[offset:]
	buf := bytes.NewReader(payload)
	server, err := DecodeGameServer(buf)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func EncodeGetGameServers(requestId *uint32) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(GetGameServers))
	var flags byte = 0x00
	payloadLength := 0
	if requestId != nil {
		payloadLength += 4
		flags |= HasIdFlag
	}
	buf.WriteByte(byte(flags))
	err := writePayloadLength(&buf, payloadLength)
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

func DecodeGetGameServers(data []byte, requestId *uint32) error {
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

func DecodeSendGameServers(data []byte, requestId *uint32) ([]*GameServerInfo, error) {
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

func EncodeSendGameServers(servers []*GameServerInfo, requestId *uint32) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(SendGameServers))
	var flags byte = 0x00
	payloadLength := 0
	if requestId != nil {
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

	writePayloadLength(&buf, payloadLength)
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

func EncodeGameServer(server *GameServerInfo) ([]byte, error) {
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

func writeStringWithLength(buf *bytes.Buffer, str string) error {
	err := writePayloadLength(buf, len(str))
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

func EncodeSpawnServerRequest(name string, requestId *uint32) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(SpawnServerRequest))
	var flag byte = 0x00
	offset := 0
	if requestId != nil {
		offset += 4
		flag |= HasIdFlag
	}
	buf.WriteByte(byte(flag))
	err := writePayloadLength(&buf, len(name)+offset)
	if requestId != nil {
		if err := binary.Write(&buf, binary.BigEndian, requestId); err != nil {
			return nil, err
		}
	}
	_, err = buf.WriteString(name)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeSpawnServerRequest(data []byte, requestId *uint32) (string, error) {
	_, err := checkAndDecodeLength(data, SpawnServerRequest)
	if err != nil {
		return "", err
	}
	offset := HeaderLength
	if requestId != nil {
		if err = GetRequestId(data, requestId); err != nil {
			return "", nil
		}
		offset += 4
	}
	payload := data[offset:]
	serverName := string(payload)
	return serverName, nil
}

func DecodeGameEnd(data []byte) (GameEndType, error) {
	_, err := checkAndDecodeLength(data, GameEnd)
	if err != nil {
		return 0, err
	}
	return GameEndType(data[HeaderLength]), nil
}

func EncodeTextMessage(message string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(TextMessage))
	buf.WriteByte(byte(0x00))
	payload := []byte(message)
	err := writePayloadLength(&buf, len(payload))
	if err != nil {
		return nil, err
	}
	buf.Write(payload)
	return buf.Bytes(), nil
}

func DecodeTextMessage(data []byte) (string, error) {
	_, err := checkAndDecodeLength(data, TextMessage)
	if err != nil {
		return "", err
	}
	payload := data[HeaderLength:]
	return string(payload), nil
}

func EncodeMove(move mines.Move) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(MoveCommand))
	// Reserved byte for future use
	buf.WriteByte(byte(0x00))
	payload := make([]byte, 13)
	payload[0] = byte(move.Type)
	copy(payload[1:5], intToBytes(move.X))
	copy(payload[5:9], intToBytes(move.Y))
	binary.BigEndian.PutUint32(payload[9:13], move.PlayerId)

	err := writePayloadLength(&buf, len(payload))
	if err != nil {
		return nil, err
	}
	buf.Write(payload)
	return buf.Bytes(), nil

}

func DecodeMove(data []byte) (move *mines.Move, err error) {
	_, err = checkAndDecodeLength(data, MoveCommand)
	if err != nil {
		return nil, err
	}
	move = &mines.Move{}
	payload := data[HeaderLength:]
	move.Type = mines.MoveType(payload[0])
	move.X = bytesToInt(payload[1:5])
	move.Y = bytesToInt(payload[5:9])
	move.PlayerId = binary.BigEndian.Uint32(payload[9:13])
	return move, nil
}

const (
	MineFlag     byte = 0b0001
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

func decodeCellFlags(flags byte, cell *mines.Cell) {
	cell.Mine = (flags & MineFlag) != 0
	cell.Revealed = (flags & RevealedFlag) != 0
	cell.Flagged = (flags & FlaggedFlag) != 0
}

func decodeCell(data []byte) (*mines.Cell, error) {
	if len(data) != CellByteLength {
		return nil, fmt.Errorf("Invalid length to decode cell (%d)", len(data))
	}
	cell := mines.Cell{}
	cell.X = bytesToInt(data[0:4])
	cell.Y = bytesToInt(data[4:8])
	flags := data[8]
	decodeCellFlags(flags, &cell)
	return &cell, nil
}

func EncodeBoard(board *mines.Board) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(Board))
	buf.WriteByte(byte(0x00))
	var boardBuf bytes.Buffer
	boardBuf.Write(intToBytes(board.Height))
	boardBuf.Write(intToBytes(board.Width))
	for y := range board.Height {
		for x := range board.Width {
			boardBuf.Write(encodeCell(board.Cells[x][y]))
		}
	}
	err := writePayloadLength(&buf, boardBuf.Len())
	if err != nil {
		return nil, err
	}
	buf.Write(boardBuf.Bytes())
	return buf.Bytes(), nil

}

func DecodeBoard(data []byte) (*mines.Board, error) {
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
	if len(cells)%CellByteLength != 0 {
		return nil, fmt.Errorf("Cells payload length mismatch")
	}
	if len(cells)/CellByteLength != board.Height*board.Width {
		return nil, fmt.Errorf("Number of cells doesnt match board size")
	}
	for i := 0; i < len(cells); i += CellByteLength {
		cell, err := decodeCell(cells[i : i+CellByteLength])
		if err != nil {
			return nil, err
		}
		if cell.X < 0 || cell.X >= board.Width || cell.Y < 0 || cell.Y >= board.Height {
			return nil, fmt.Errorf("Cell position out of bounds: (%d, %d)", cell.X, cell.Y)
		}
		if board.Cells[cell.X][cell.Y] != nil {
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
	err := writePayloadLength(&buf, payloadLength)
	if err != nil {
		return nil, err
	}
	for _, cell := range cells {
		buf.Write(encodeCellUpdate(cell))
	}
	if payloadLength+HeaderLength != buf.Len() {
		return nil, fmt.Errorf("Incorrect payload length while encoding cell updates")
	}
	return buf.Bytes(), nil
}

func decodeCellUpdate(data []byte) (*mines.UpdatedCell, error) {
	if len(data) != UpdateCellByteLength {
		return nil, fmt.Errorf("incorrect byte length to decode cell update (%d)", len(data))
	}
	cell := &mines.UpdatedCell{
		X:     bytesToInt(data[0:4]),
		Y:     bytesToInt(data[4:8]),
		Value: data[8]}
	return cell, nil

}

func DecodeCellUpdates(data []byte) ([]mines.UpdatedCell, error) {

	payloadLength, err := checkAndDecodeLength(data, CellUpdate)
	if err != nil {
		return nil, err
	}
	payload := data[HeaderLength:]
	if payloadLength%UpdateCellByteLength != 0 {
		return nil, fmt.Errorf("update cells payload length mismatch %d", payloadLength)
	}
	cells := make([]mines.UpdatedCell, payloadLength/UpdateCellByteLength)
	for i := range payloadLength / UpdateCellByteLength {
		cell, err := decodeCellUpdate(payload[i*UpdateCellByteLength : (i+1)*UpdateCellByteLength])
		if err != nil {
			return nil, err
		}
		cells[i] = *cell
	}
	return cells, nil
}

func EncodeGamemodeInfo(info mines.GamemodeUpdateInfo) ([]byte, error) {
	switch info.GetGameModeId() {
	case mines.ModeCoop:
		i, ok := info.(*mines.CoopInfoUpdate)
		if !ok {
			return nil, fmt.Errorf("Failed to cast to CoopInfoUpdate")
		}
		return EncodeCoopInfoUpdate(i)
	default:
		return nil, fmt.Errorf("Gamemode info not implemented to decode")
	}
}

func DecodeGamemodeInfo(data []byte) (mines.GamemodeUpdateInfo, error) {
	_, err := checkAndDecodeLength(data, GamemodeInfo)
	if err != nil {
		return nil, err
	}
	gamemodeId := mines.GameModeId(data[HeaderLength])
	switch gamemodeId {
	case mines.ModeCoop:
		return DecodeCoopInfoUpdate(data)
	default:
		return nil, fmt.Errorf("Can't decode gamemode info gamemodeId: %d", gamemodeId)
	}

}
func DecodeCoopInfoUpdate(data []byte) (*mines.CoopInfoUpdate, error) {
	if data[HeaderLength] != byte(mines.ModeCoop) {
		return nil, fmt.Errorf("Wrong gamemodeId to decode Coop: %d", data[4])
	}
	offset := HeaderLength + 1
	scoreLength := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2
	playerScores := make(map[uint32]int)
	for range scoreLength {
		playerId := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
		score := bytesToInt(data[offset : offset+4])
		playerScores[playerId] = score
		offset += 4
	}
	var marksChange []mines.PlayerMarkChange
	for offset < len(data) {
		X := bytesToInt(data[offset : offset+4])
		offset += 4
		Y := bytesToInt(data[offset : offset+4])
		offset += 4
		playerId := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
		marksChange = append(marksChange, mines.PlayerMarkChange{X: X, Y: Y, PlayerId: playerId})
	}

	return &mines.CoopInfoUpdate{MarksChange: marksChange, PlayerScores: playerScores}, nil

}

func EncodeCoopInfoUpdate(info *mines.CoopInfoUpdate) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(byte(GamemodeInfo))
	buf.WriteByte(byte(0x00))
	// |gamemodeId + len(playesScores) + size(marksChange) + size(playersCores)|
	payloadLength := 1 + 2 + 12*len(info.MarksChange) + 8*len(info.PlayerScores)
	err := writePayloadLength(&buf, payloadLength)

	buf.WriteByte(byte(mines.ModeCoop))
	binary.Write(&buf, binary.BigEndian, uint16(len(info.PlayerScores)))
	for playerId, score := range info.PlayerScores {
		binary.Write(&buf, binary.BigEndian, playerId)
		binary.Write(&buf, binary.BigEndian, uint32(score))
	}
	for _, cellInfo := range info.MarksChange {
		binary.Write(&buf, binary.BigEndian, uint32(cellInfo.X))
		binary.Write(&buf, binary.BigEndian, uint32(cellInfo.Y))
		binary.Write(&buf, binary.BigEndian, cellInfo.PlayerId)
	}
	return buf.Bytes(), err
}

func EncodeGameStart(params mines.GameParams) ([]byte, error) {
	payloadLength := 3*4 + 1
	var buf bytes.Buffer
	buf.WriteByte(byte(StartGame))
	buf.WriteByte(byte(0x00))
	err := writePayloadLength(&buf, payloadLength)
	if err != nil {
		return nil, err
	}
	payload := make([]byte, payloadLength)
	copy(payload[0:4], intToBytes(params.Width))
	copy(payload[4:8], intToBytes(params.Height))
	copy(payload[8:12], intToBytes(params.Mines))
	payload[12] = byte(params.GameMode)
	buf.Write(payload)
	return buf.Bytes(), nil

}

func DecodeGameStart(data []byte) (*mines.GameParams, error) {
	payloadLength, err := checkAndDecodeLength(data, StartGame)
	if err != nil {
		return nil, err
	}
	if payloadLength != 3*4+1 {
		return nil, fmt.Errorf("decode game starte payload incorrect length (%d)", payloadLength)
	}
	payload := data[HeaderLength:]
	params := &mines.GameParams{Width: bytesToInt(payload[0:4]),
		Height:   bytesToInt(payload[4:8]),
		Mines:    bytesToInt(payload[8:12]),
		GameMode: mines.GameModeId(payload[12]),
	}
	return params, nil
}
