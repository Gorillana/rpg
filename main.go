// inspired by Jack Mott on Youtube's GamewithGo series, specifically his RPG videos
// installed SDL dll files locally on machine
	// if not working on your machine, you may have to install the dll files for sdl2, img, mix, ttf locally
	// 	"github.com/veandco/go-sdl2/mix"
	//  "github.com/veandco/go-sdl2/sdl"
	//  "github.com/veandco/go-sdl2/ttf"

package main

import (
	"runtime"

	"github.com/gorillana/rpg/game"
	"github.com/gorillana/rpg/ui2d"
)

// Windows and Linux machines
func main() {
	game := game.NewGame(1)

	for i := 0; i < 1; i++ {
		go func(i int) {
			// calls LockOSThread inside go routine in order to keep the sdl code called in one thread
			runtime.LockOSThread()
			ui := ui2d.NewUI(game.InputChan, game.LevelChans[i])
			ui.Run()
		}(i)
	}
	game.Run()
}

// Mac machines
// func main() {
//game := game.NewGame(1, "game/maps/level1.map")
//go func() {
//	game.Run()
//}()
// ui := ui2d.NewUI(game.InputChan, game.LevelChans[0])
// ui.Run()
//}
