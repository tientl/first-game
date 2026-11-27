package main

import (
	"flag"
	"strings"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
)

// The HUD, for checking its layout:
//
//	shot -status cold,poison -skillstatus stun,bleed -buffs   status chips, a long line just begun
//	shot -notices "one|two|three"                             three notices stacked
var (
	skillStatusFlag = flag.String("skillstatus", "", "what the beasts' moves leave on the player, comma-separated: stun, bleed, slow, pinned")
	noticesFlag     = flag.String("notices", "", "notices on the screen at once, separated by |, the last newest")
)

// hudScene changes snap as the HUD's flags ask.
func hudScene(snap *protocol.Snapshot) {
	for _, st := range strings.Split(*skillStatusFlag, ",") {
		switch st {
		case "stun":
			snap.Self.Stunned = true
		case "bleed":
			snap.Self.Bleeding = true
		case "slow":
			snap.Self.Slowed = true
		case "pinned":
			snap.Self.Pinned = true
		}
	}
}

// hudGame sets the game up as the HUD's flags ask.
func hudGame(g *client.Game) {
	if *noticesFlag != "" {
		g.ShowNotices(strings.Split(*noticesFlag, "|")...)
	}
}
