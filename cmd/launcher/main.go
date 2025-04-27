package main

import (
	"fmt"

	"github.com/tomasstrnad1997/mines/gamelauncher"
)
func main() {
	launcher, err := gamelauncher.CreateGameLauncher("localhost", 42070)
	if err != nil {
		println("Failed to launch game launcher")
	}
	println("GameLauncher running...")
	go launcher.ManageCommands()
	for i := range 5 {
		launcher.SpawnNewGameServer(fmt.Sprintf("Server %d", i))
	}
	launcher.Loop()
}
