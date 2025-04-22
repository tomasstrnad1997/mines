package main

import (
	"github.com/tomasstrnad1997/mines/gamelauncher"
)
func main() {
	launcher, err := gamelauncher.CreateGameLauncher("mines.strnadt.cz", 42070)
	if err != nil {
		println("Failed to launch game launcher")
	}
	println("GameLauncher running...")
	go launcher.ManageCommands()
	launcher.Loop()
}
