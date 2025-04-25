package protocol_test

import (
	"bytes"
	"testing"

	"github.com/tomasstrnad1997/mines/protocol"
)



func TestServerInfoEncoding(t *testing.T){
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

func TestServerInfoMessageEncoding(t *testing.T){
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
