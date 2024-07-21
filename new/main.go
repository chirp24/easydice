package main

import (
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/in"
	"xabbo.b7c.io/goearth/shockwave/out"
	"xabbo.b7c.io/goearth/shockwave/profile"
	"xabbo.b7c.io/goearth/shockwave/room"
)

var ext = g.NewExt(g.ExtInfo{
	Title:       "easydice",
	Description: "An extension to roll/reset dice for you.",
	Author:      "chirp24",
	Version:     "1.6",
})

var (
	setup       bool
	setupMutex  sync.Mutex
	roomMgr     = room.NewManager(ext)
	profileMgr  = profile.NewManager(ext)
	diceIDs     []Dice
	diceIDMutex sync.Mutex
)

// Dice is a wrapper for sorting dice by their X-coordinate and including their position
type Dice struct {
	ID string
	X  int
	Y  int
}

func main() {
	ext.Initialized(onInitialized)
	ext.Connected(onConnected)
	ext.Disconnected(onDisconnected)
	ext.Intercept(out.CHAT, out.SHOUT).With(handleChat)

	ext.Run()
}

func onInitialized(e g.InitArgs) {
	log.Println("Extension initialized")
}

func onConnected(e g.ConnectArgs) {
	log.Printf("Game connected (%s)\n", e.Host)
}

func onDisconnected() {
	log.Println("Game disconnected")
}

func showMsg(msg string) {
	self := roomMgr.EntityByName(profileMgr.Name)
	if self == nil {
		return
	}
	ext.Send(in.CHAT, self.Index, msg)
}

func handleChat(e *g.Intercept) {
	msg := e.Packet.ReadString()
	if strings.Contains(msg, ":close") {
		e.Block()
		log.Println(msg)
		setupMutex.Lock()
		setup = false
		setupMutex.Unlock()
		go closeDice()
	} else if strings.Contains(msg, ":setup") {
		e.Block()
		go showMsg("Setup mode enabled.")
		log.Println(msg)
		setupMutex.Lock()
		setup = true
		setupMutex.Unlock()
		go collectDice()
	} else if strings.Contains(msg, ":roll") {
		e.Block()
		log.Println(msg)
		go rollDice()
	}
}

func filterDice(roomMgr *room.Manager, self *room.Entity) []room.Object {
	var dice []room.Object
	for _, obj := range roomMgr.Objects {
		if !strings.HasPrefix(obj.Class, "edice") {
			continue
		}
		dx, dy := self.X-obj.X, self.Y-obj.Y
		if (dx >= -1 && dx <= 1) && (dy >= -1 && dy <= 1) {
			dice = append(dice, obj)
		}
	}
	return dice
}

func collectDice() {
	self := roomMgr.EntityByName(profileMgr.Name)
	if self == nil {
		log.Println("self not found.")
		return
	}

	log.Println("Starting dice collection")

	dice := filterDice(roomMgr, self)
	log.Println("Filtered dice:", dice)

	var diceList []Dice
	for _, die := range dice {
		diceList = append(diceList, Dice{ID: die.Id, X: die.X, Y: die.Y})
	}

	layout := detectLayout(diceList, self)
	log.Printf("Detected layout: %s", layout)
	if layout == "unknown" {
		go showMsg("Unknown dice layout.")
		return
	}

	switch layout {
	case "bottom":
		sortDice(diceList, self)
	case "top":
		sortDice(diceList, self)
	case "left":
		sortDice(diceList, self)
	case "right":
		sortDice(diceList, self)
	default:
		log.Println("Unknown layout detected.")
		return
	}

	diceIDMutex.Lock()
	defer diceIDMutex.Unlock()

	if setup {
		diceIDs = nil
		for _, die := range diceList {
			if len(diceIDs) < 5 {
				diceIDs = append(diceIDs, die)
				log.Printf("Collected dice ID: %s (X: %d, Y: %d)", die.ID, die.X, die.Y)
				if len(diceIDs) == 5 {
					setupMutex.Lock()
					setup = false
					setupMutex.Unlock()
					go showMsg("Setup mode disabled. 5 dice IDs collected.")
					break
				}
			}
		}
	}
}

func detectLayout(diceList []Dice, self *room.Entity) string {
	xSet := make(map[int]bool)
	ySet := make(map[int]bool)

	for _, die := range diceList {
		xSet[die.X] = true
		ySet[die.Y] = true
	}

	xCoords := make([]int, 0, len(xSet))
	yCoords := make([]int, 0, len(ySet))
	for x := range xSet {
		xCoords = append(xCoords, x)
	}
	for y := range ySet {
		yCoords = append(yCoords, y)
	}
	sort.Ints(xCoords)
	sort.Ints(yCoords)

	log.Printf("Player coordinates: X=%d, Y=%d", self.X, self.Y)
	log.Printf("xCoords: %v, yCoords: %v", xCoords, yCoords)

	if len(ySet) == 3 && len(xSet) == 2 && xSet[self.X] && xSet[self.X+1] {
		if ySet[self.Y-1] && ySet[self.Y] && ySet[self.Y+1] {
			return "bottom"
		}
	}

	if len(ySet) == 3 && len(xSet) == 2 && xSet[self.X] && xSet[self.X-1] {
		if ySet[self.Y-1] && ySet[self.Y] && ySet[self.Y+1] {
			return "top"
		}
	}

	if len(xSet) == 3 && len(ySet) == 2 && ySet[self.Y] && ySet[self.Y+1] {
		if xSet[self.X-1] && xSet[self.X] && xSet[self.X+1] {
			return "left"
		}
	}

	if len(xSet) == 3 && len(ySet) == 2 && ySet[self.Y] && ySet[self.Y-1] {
		if xSet[self.X-1] && xSet[self.X] && xSet[self.X+1] {
			return "right"
		}
	}

	return "unknown"
}

func sortDice(diceList []Dice, self *room.Entity) {
	sort.Slice(diceList, func(i, j int) bool {
		return angleFromCenter(diceList[i], self) < angleFromCenter(diceList[j], self)
	})
}

func angleFromCenter(dice Dice, self *room.Entity) float64 {
	dx := float64(dice.X - self.X)
	dy := float64(dice.Y - self.Y)
	return math.Atan2(dy, dx)
}

func closeDice() {
	diceIDMutex.Lock()
	defer diceIDMutex.Unlock()

	for _, die := range diceIDs {
		ext.Send(out.DICE_OFF, []byte(die.ID))
		time.Sleep(1 * time.Second)
	}
}

func rollDice() {
	diceIDMutex.Lock()
	defer diceIDMutex.Unlock()

	for _, die := range diceIDs {
		ext.Send(out.THROW_DICE, []byte(die.ID))
		time.Sleep(1 * time.Second)
	}
}
