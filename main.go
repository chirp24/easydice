package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/in"
	"xabbo.b7c.io/goearth/shockwave/out"
)

var ext = g.NewExt(g.ExtInfo{
	Title:       "easydice",
	Description: "An extension to roll/reset dice for you.",
	Author:      "chirp24",
	Version:     "1.3",
})

var (
	setup          bool
	setupMutex     sync.Mutex
	diceIDs        []int
	diceValues     map[int]int
	throwChannel   chan *g.Packet
	diceOffChannel chan *g.Packet
)

func main() {
	ext.Initialized(onInitialized)
	ext.Connected(onConnected)
	ext.Disconnected(onDisconnected)
	ext.Intercept(out.CHAT, out.SHOUT, out.WHISPER).With(handleChat)
	ext.Intercept(out.THROW_DICE).With(handleThrowDice)
	ext.Intercept(out.DICE_OFF).With(handleDiceOff)
	ext.Intercept(in.DICE_VALUE).With(handleDiceResults)

	throwChannel = make(chan *g.Packet, 5)
	diceOffChannel = make(chan *g.Packet, 5)
	diceValues = make(map[int]int)

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
		ext.Send(in.CHAT, 0, "> > Setup mode enabled. Roll/Close your five dice. < <", 0, 34, 0, 0)
		setup = true
		setupMutex.Unlock()
		// Reset all saved packets
		resetPackets()
	} else if strings.Contains(msg, ":roll") { // :roll msg
		e.Block()
		log.Println(msg)
		go rollDice()
	} else if strings.Contains(msg, ":tri") { // :tri msg
		e.Block()
		log.Println(msg)
		go rollSpecificDice([]int{0, 2, 4}) // Roll 1st, 3rd, and 5th dice IDs
	}
}

func rollSpecificDice(ids []int) {
	done := make(chan struct{})

	for _, id := range ids {
		go func(diceID int) {
			packet := fmt.Sprintf("%d", diceID)      // Construct dice roll packet
			ext.Send(out.THROW_DICE, []byte(packet)) // Send the packet
			log.Printf("Sent dice roll packet for ID: %d\n", diceID)
			done <- struct{}{}
		}(diceIDs[id])

		time.Sleep(550 * time.Millisecond)
	}

	for range ids {
		<-done
	}

	log.Println("All specific dice roll packets sent")
}

func resetPackets() {
	setupMutex.Lock()
	defer setupMutex.Unlock()
	diceIDs = nil
	diceValues = make(map[int]int)
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
		if len(diceIDs) < 5 {
			packetString := string(packet.Data)
			diceID, err := strconv.Atoi(packetString)
			if err != nil {
				log.Printf("Error parsing dice ID: %v\n", err)
			} else {
				diceIDs = append(diceIDs, diceID)
				log.Printf("Dice ID captured: %d\n", diceID)
			}
		}
		setupMutex.Unlock()
	}
	log.Println("Throw Dice Setup complete")
}

func handleDiceOffSetup() {
	for packet := range diceOffChannel {
		// Process dice off packets if needed
		setupMutex.Lock()
		packetString := string(packet.Data)
		diceID, err := strconv.Atoi(packetString)
		if err != nil {
			log.Printf("Error parsing dice ID: %v\n", err)
		} else {
			diceIDs = append(diceIDs, diceID)
			log.Printf("Dice ID captured: %d\n", diceID)
		}
		setupMutex.Unlock()
	}
	log.Println("Dice Off Setup complete")
}

func closeDice() {
	done := make(chan struct{})

	for _, id := range diceIDs {
		go func(diceID int) {
			packet := fmt.Sprintf("%d", diceID)    // Construct dice off packet
			ext.Send(out.DICE_OFF, []byte(packet)) // Send the packet
			log.Printf("Sent dice close packet for ID: %d\n", diceID)
			done <- struct{}{}
		}(id)

		time.Sleep(550 * time.Millisecond)
	}

	for range diceIDs {
		<-done
	}

	log.Println("All dice close packets sent")
}

func rollDice() {
	done := make(chan struct{})

	for _, id := range diceIDs {
		go func(diceID int) {
			packet := fmt.Sprintf("%d", diceID)      // Construct dice roll packet
			ext.Send(out.THROW_DICE, []byte(packet)) // Send the packet
			log.Printf("Sent dice roll packet for ID: %d\n", diceID)
			done <- struct{}{}
		}(id)

		time.Sleep(5 * time.Millisecond)
	}

	for range diceIDs {
		<-done
	}

	log.Println("All dice roll packets sent")
}

//func parseDiceValuePacket(packet *g.Packet) (int, int, error) {
//	packetString := string(packet.Data) // Convert packet data to string
//	parts := strings.Fields(packetString)
//	if len(parts) < 2 {
//		return 0, 0, fmt.Errorf("packet format incorrect")
//	}
//
//	diceID, err := strconv.Atoi(parts[0])
//	if err != nil {
//		return 0, 0, fmt.Errorf("error parsing dice ID: %v", err)
//	}
//
//	obfuscatedValue, err := strconv.Atoi(parts[1])
//	if err != nil {
//		return 0, 0, fmt.Errorf("error parsing obfuscated value: %v", err)
//	}
//
//	return diceID, obfuscatedValue, nil
//}

func handleDiceResults(e *g.Intercept) {
	packetString := string(e.Packet.Data)
	parts := strings.Fields(packetString)

	if len(parts) != 2 {
		log.Println("Invalid DICE_VALUE packet format. Expected two parts.")
		return
	}

	diceID, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Printf("Error parsing dice ID: %v\n", err)
		return
	}

	obfuscatedValue, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Printf("Error parsing obfuscated value: %v\n", err)
		return
	}

	setupMutex.Lock()
	defer setupMutex.Unlock()

	for i, id := range diceIDs {
		if diceID == id {
			diceValues[i] = obfuscatedValue - id*38
			break
		}
	}

	// Check if we have all dice values now
	if len(diceValues) == 5 {
		message := evaluateDiceValues(diceValues)
		ext.Send(in.CHAT, 0, message, 0, 34, 0, 0) // Sending the result as a chat message
		diceValues = make(map[int]int)             // Reset for next roll
	}
}

func evaluateDiceValues(diceValues map[int]int) string {
	// Check for any dice value being 0
	for _, value := range diceValues {
		if value == 0 {
			return "" // Return empty string to indicate void hand
		}
	}

	// Check for a straight (1,2,3,4,5 or 2,3,4,5,6)
	straightValues1 := []int{1, 2, 3, 4, 5}
	straightValues2 := []int{2, 3, 4, 5, 6}

	straight1 := true
	straight2 := true

	// Check if all required values are present in diceValues
	for _, value := range straightValues1 {
		if _, ok := diceValues[value]; !ok {
			straight1 = false
			break
		}
	}

	for _, value := range straightValues2 {
		if _, ok := diceValues[value]; !ok {
			straight2 = false
			break
		}
	}

	// If straight is true for either pattern, return the appropriate message
	if straight1 || straight2 {
		return "Straight! (this message is only seen by you!)"
	}

	// Check for other hands based on counts
	var counts = make(map[int]int)
	for _, value := range diceValues {
		counts[value]++
	}

	switch len(counts) {
	case 4:
		return "One pair! (this message is only seen by you!)"
	case 3:
		for _, count := range counts {
			if count == 3 {
				return "Three of a kind! (this message is only seen by you!)"
			}
		}
		return "Two pair! (this message is only seen by you!)"
	case 2:
		for _, count := range counts {
			if count == 4 {
				return "Four of a kind! (this message is only seen by you!)"
			}
		}
		return "Full house! (this message is only seen by you!)"
	default:
		return "Invalid hand! (this message is only seen by you!)"
	}
}
