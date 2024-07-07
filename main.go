package main

import (
	"log"
	"strings"
	"sync"
	"time"

	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/out"
)

var ext = g.NewExt(g.ExtInfo{
	Title:       "easydice",
	Description: "An extension to roll/reset dice for you.",
	Author:      "chirp24",
	Version:     "1.0",
})

var (
	throwDice1     *g.Packet
	throwDice2     *g.Packet
	throwDice3     *g.Packet
	throwDice4     *g.Packet
	throwDice5     *g.Packet
	diceOff1       *g.Packet
	diceOff2       *g.Packet
	diceOff3       *g.Packet
	diceOff4       *g.Packet
	diceOff5       *g.Packet
	setup          bool
	throwChannel   chan *g.Packet
	diceOffChannel chan *g.Packet
	setupMutex     sync.Mutex
)

func main() {
	ext.Initialized(onInitialized)
	ext.Connected(onConnected)
	ext.Disconnected(onDisconnected)
	ext.Intercept(out.CHAT, out.SHOUT, out.WHISPER).With(handleChat)
	ext.Intercept(out.THROW_DICE).With(handleThrowDice)
	ext.Intercept(out.DICE_OFF).With(handleDiceOff)

	throwChannel = make(chan *g.Packet, 5)
	diceOffChannel = make(chan *g.Packet, 5)

	go handleThrowDiceSetup()
	go handleDiceOffSetup()

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
		log.Println(msg)
		setupMutex.Lock()
		setup = true
		setupMutex.Unlock()
		// Reset all saved packets
		resetPackets()
	} else if strings.Contains(msg, ":roll") { // :roll msg
		e.Block()
		log.Println(msg)
		go rollDice()
	}
}

func resetPackets() {
	setupMutex.Lock()
	defer setupMutex.Unlock()
	throwDice1, throwDice2, throwDice3, throwDice4, throwDice5 = nil, nil, nil, nil, nil
	diceOff1, diceOff2, diceOff3, diceOff4, diceOff5 = nil, nil, nil, nil, nil
	log.Println("All saved packets reset")
}

func handleThrowDice(e *g.Intercept) {
	setupMutex.Lock()
	defer setupMutex.Unlock()
	if setup {
		throwChannel <- e.Packet.Copy()
	}
}

func handleDiceOff(e *g.Intercept) {
	setupMutex.Lock()
	defer setupMutex.Unlock()
	if setup {
		diceOffChannel <- e.Packet.Copy()
	}
}

func handleThrowDiceSetup() {
	for packet := range throwChannel {
		setupMutex.Lock()
		if throwDice1 == nil {
			throwDice1 = packet
			log.Println("Throw Dice 1 set:", throwDice1)
		} else if throwDice2 == nil {
			throwDice2 = packet
			log.Println("Throw Dice 2 set:", throwDice2)
		} else if throwDice3 == nil {
			throwDice3 = packet
			log.Println("Throw Dice 3 set:", throwDice3)
		} else if throwDice4 == nil {
			throwDice4 = packet
			log.Println("Throw Dice 4 set:", throwDice4)
		} else if throwDice5 == nil {
			throwDice5 = packet
			log.Println("Throw Dice 5 set:", throwDice5)
		}
		setupMutex.Unlock()
	}
	log.Println("Throw Dice Setup complete")
}

func handleDiceOffSetup() {
	for packet := range diceOffChannel {
		setupMutex.Lock()
		if diceOff1 == nil {
			diceOff1 = packet
			log.Println("Dice Off 1 set:", diceOff1)
		} else if diceOff2 == nil {
			diceOff2 = packet
			log.Println("Dice Off 2 set:", diceOff2)
		} else if diceOff3 == nil {
			diceOff3 = packet
			log.Println("Dice Off 3 set:", diceOff3)
		} else if diceOff4 == nil {
			diceOff4 = packet
			log.Println("Dice Off 4 set:", diceOff4)
		} else if diceOff5 == nil {
			diceOff5 = packet
			log.Println("Dice Off 5 set:", diceOff5)
		}
		setupMutex.Unlock()
	}
	log.Println("Dice Off Setup complete")
}

func closeDice() {
	setupMutex.Lock()
	defer setupMutex.Unlock()
	if diceOff1 != nil {
		ext.SendPacket(diceOff1)
		time.Sleep(500 * time.Millisecond)
	}
	if diceOff2 != nil {
		ext.SendPacket(diceOff2)
		time.Sleep(500 * time.Millisecond)
	}
	if diceOff3 != nil {
		ext.SendPacket(diceOff3)
		time.Sleep(500 * time.Millisecond)
	}
	if diceOff4 != nil {
		ext.SendPacket(diceOff4)
		time.Sleep(500 * time.Millisecond)
	}
	if diceOff5 != nil {
		ext.SendPacket(diceOff5)
	}
}

func rollDice() {
	setupMutex.Lock()
	defer setupMutex.Unlock()

	if throwDice1 != nil {
		ext.SendPacket(throwDice1)
		time.Sleep(500 * time.Millisecond)
	}
	if throwDice2 != nil {
		ext.SendPacket(throwDice2)
		time.Sleep(500 * time.Millisecond)
	}
	if throwDice3 != nil {
		ext.SendPacket(throwDice3)
		time.Sleep(500 * time.Millisecond)
	}
	if throwDice4 != nil {
		ext.SendPacket(throwDice4)
		time.Sleep(500 * time.Millisecond)
	}
	if throwDice5 != nil {
		ext.SendPacket(throwDice5)
	}
}
