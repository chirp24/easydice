package main

import (
	"log"
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
	if strings.Contains(msg, ":close") { // :close msg
		e.Block()
		log.Println(msg)
		setupMutex.Lock()
		setup = false
		setupMutex.Unlock()
		go closeDice()
	} else if strings.Contains(msg, ":setup") { // :setup msg
		e.Block()
		go showMsg("Setup mode enabled.")
		log.Println(msg)
		setupMutex.Lock()
		setup = true
		setupMutex.Unlock()
		go collectDice()
	} else if strings.Contains(msg, ":roll") { // :roll msg
		e.Block()
		log.Println(msg)
		go rollDice()
	} else if strings.Contains(msg, ":tri") { // :tri msg
		e.Block()
		log.Println(msg)
		go rollTriangle()
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

	// Prepare for sorting
	var diceList []Dice
	for _, die := range dice {
		diceList = append(diceList, Dice{ID: die.Id, X: die.X, Y: die.Y})
	}

	if len(diceList) != 5 {
		log.Println("Expected 5 dice but found:", len(diceList))
		return
	}

	// Determine layout and sort dice accordingly
	layout := detectLayout(diceList, self)
	switch layout {
	case "below":
		log.Println("Detected dice layout: below")
		sortDiceBelow(diceList)
	case "above":
		log.Println("Detected dice layout: above")
		sortDiceAbove(diceList)
	case "left":
		log.Println("Detected dice layout: left")
		sortDiceLeft(diceList)
	case "right":
		log.Println("Detected dice layout: right")
		sortDiceRight(diceList)
	default:
		log.Println("Unknown layout detected.")
		return
	}

	// Update diceIDs with sorted dice, but only if setup mode is active
	diceIDMutex.Lock()
	defer diceIDMutex.Unlock()

	if setup {
		diceIDs = nil // Clear existing diceIDs if setup mode is reactivated
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
	var xCoords, yCoords []int

	for _, die := range diceList {
		xCoords = append(xCoords, die.X)
		yCoords = append(yCoords, die.Y)
	}

	sort.Ints(xCoords)
	sort.Ints(yCoords)

	xSet := make(map[int]struct{})
	ySet := make(map[int]struct{})

	for _, x := range xCoords {
		xSet[x] = struct{}{}
	}

	for _, y := range yCoords {
		ySet[y] = struct{}{}
	}

	if len(xSet) == 2 && len(ySet) == 5 {
		// Check if the dice are to the left or right of the player
		if xCoords[0] == self.X-1 && xCoords[1] == self.X && yCoords[0] == self.Y {
			return "left"
		} else if xCoords[0] == self.X && xCoords[1] == self.X+1 && yCoords[0] == self.Y {
			return "right"
		}
	} else if len(xSet) == 5 && len(ySet) == 2 {
		// Check if the dice are below or above the player
		if yCoords[0] == self.Y-1 && yCoords[1] == self.Y && xCoords[0] == self.X {
			return "below"
		} else if yCoords[0] == self.Y+1 && yCoords[1] == self.Y && xCoords[0] == self.X {
			return "above"
		}
	} else if len(xSet) == 2 && len(ySet) == 2 {
		// Check for unexpected patterns
		return "unknown"
	}

	return "unknown"
}

func sortDiceBelow(diceList []Dice) {
	sort.Slice(diceList, func(i, j int) bool {
		if diceList[i].Y == diceList[j].Y {
			return diceList[i].X < diceList[j].X
		}
		return diceList[i].Y < diceList[j].Y
	})
}

func sortDiceAbove(diceList []Dice) {
	sort.Slice(diceList, func(i, j int) bool {
		if diceList[i].Y == diceList[j].Y {
			return diceList[i].X > diceList[j].X
		}
		return diceList[i].Y < diceList[j].Y
	})
}

func sortDiceLeft(diceList []Dice) {
	sort.Slice(diceList, func(i, j int) bool {
		if diceList[i].X == diceList[j].X {
			return diceList[i].Y < diceList[j].Y
		}
		return diceList[i].X > diceList[j].X
	})
}

func sortDiceRight(diceList []Dice) {
	sort.Slice(diceList, func(i, j int) bool {
		if diceList[i].X == diceList[j].X {
			return diceList[i].Y > diceList[j].Y
		}
		return diceList[i].X < diceList[j].X
	})
}

func closeDice() {
	diceIDMutex.Lock()
	defer diceIDMutex.Unlock()

	for _, dice := range diceIDs {
		log.Printf("Sending DICE_OFF for ID: %s (X: %d, Y: %d)", dice.ID, dice.X, dice.Y)
		ext.Send(out.DICE_OFF, []byte(dice.ID))
		time.Sleep(500 * time.Millisecond)
	}
}

func rollDice() {
	diceIDMutex.Lock()
	defer diceIDMutex.Unlock()

	for _, dice := range diceIDs {
		log.Printf("Sending THROW_DICE for ID: %s (X: %d, Y: %d)", dice.ID, dice.X, dice.Y)
		ext.Send(out.THROW_DICE, []byte(dice.ID))
		time.Sleep(500 * time.Millisecond)
	}
}

func rollTriangle() {
	diceIDMutex.Lock()
	defer diceIDMutex.Unlock()

	if len(diceIDs) < 3 {
		log.Println("Not enough dice to roll in triangle")
		return
	}

	// Select every other die (1st, 3rd, and 5th in the sorted list)
	triangleDice := []Dice{}
	for i, dice := range diceIDs {
		if i%2 == 0 { // Every other die
			triangleDice = append(triangleDice, dice)
		}
		if len(triangleDice) == 3 {
			break
		}
	}

	if len(triangleDice) < 3 {
		log.Println("Not enough dice to roll in a triangle")
		return
	}

	for _, dice := range triangleDice {
		log.Printf("Sending THROW_DICE for ID: %s (X: %d, Y: %d)", dice.ID, dice.X, dice.Y)
		ext.Send(out.THROW_DICE, []byte(dice.ID))
		time.Sleep(500 * time.Millisecond)
	}
}
