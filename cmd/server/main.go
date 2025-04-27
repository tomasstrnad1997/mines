package main

import (
	"fmt"

	"github.com/tomasstrnad1997/mines/server"
)
func main() {
	server, err := server.SpawnServer(0, "Server", 42069)
	if err != nil {
		println("Failed to start server")
		return
	}
	fmt.Printf("%s started at %d\n", server.Name, server.Port)
	for {}
}
