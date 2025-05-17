package protocol_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/tomasstrnad1997/mines/players"
	"github.com/tomasstrnad1997/mines/protocol"
)

func TestServerInfoEncoding(t *testing.T) {
	info := &protocol.GameServerInfo{"Game server 69", "127.0.0.1", 42069, 3}
	encoded, err := protocol.EncodeGameServer(info)
	if err != nil {
		t.Fatalf("Failed to encode game info: %v", err)
	}
	buf := bytes.NewReader(encoded)
	decoded, err := protocol.DecodeGameServer(buf)
	if err != nil {
		t.Fatalf("Failed to decode game info: %v", err)
	}
	if *decoded != *info {
		t.Fatalf("Decoded does not match original")
	}

}

func TestServerInfoMessageEncoding(t *testing.T) {
	servers := []*protocol.GameServerInfo{
		{"Game server 69", "127.0.0.1", 42069, 3},
		{"GS Rest", "192.168.0.1", 11111, 7},
		{"FD Free", "10.0.0.5", 429, 0},
	}

	encoded, err := protocol.EncodeSendGameServers(servers, nil)
	if err != nil {
		t.Fatalf("Failed to encode game servers info: %v", err)
	}
	decoded, err := protocol.DecodeSendGameServers(encoded, nil)
	if err != nil {
		t.Fatalf("Failed to decode game servers info: %v", err)
	}
	for i, server := range decoded {
		if *server != *servers[i] {
			t.Fatalf("Decoded game servers do not match original")
		}
	}
}

func TestGameConnectionResponse(t *testing.T) {
	token := players.AuthToken{
		PlayerID: 1235,
		Expiry:   time.Now().Unix(),
		Nonce:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Signature: [32]byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0xf7, 0x18,
			0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0xf0,
		},
	}
	gsInfo := protocol.GameServerConnectInfo{
		Host: "test.com",
		Port: 420,
	}
	original := protocol.GameConnectionResponse{
		Success:  true,
		GameInfo: &gsInfo,
		Token:    &token,
	}

	encoded, err := protocol.EncodeConnectToGameResponse(original)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	decoded, err := protocol.DecodeConnectToGameResponse(encoded)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if original.Success != decoded.Success {
		t.Fatalf("success doesn't match")
	}
	if token != *decoded.Token {
		t.Fatalf("Token doesn't match")
	}
	if gsInfo != *decoded.GameInfo {
		t.Fatalf("Game info doesn't match")
	}
}
func TestGameConnectionResponseDeny(t *testing.T) {
	original := protocol.GameConnectionResponse{Success: false}

	encoded, err := protocol.EncodeConnectToGameResponse(original)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	decoded, err := protocol.DecodeConnectToGameResponse(encoded)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if original.Success != decoded.Success {
		t.Fatalf("success doesn't match")
	}
}
