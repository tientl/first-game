package main

import (
	"survivalgame/internal/client"
	"survivalgame/internal/sim"
)

// devShotLines are the commands the -devconsole shot shows, run for real
// on the world so the answers are the game's own.
var devShotLines = []string{"help", "give gỗ 20", "give \"rìu đá lửa\"", "spawn trike 3", "tp làng", "weather storm", "list beasts khung long", "god"}

// showDevConsole opens the dev console on game with devShotLines and what
// the world answered.
func showDevConsole(game *client.Game, w *sim.World) {
	id := w.AddPlayer("Dev")
	var answers [][]string
	for _, l := range devShotLines {
		answers = append(answers, w.DevCommand(id, l).Lines)
	}
	game.ShowDevConsole(devShotLines, answers)
}
