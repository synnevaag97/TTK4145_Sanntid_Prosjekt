package main

import (
	// "Driver-go/elevio"
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"fmt"
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//  will be received as zero-values.
/*
type HelloMsg struct {
	Message string
	Iter    int
}
*/

var LocalID string

var SingleElevatorMode bool = false
var counter int = 0

type MessageType int

const (
	MT_Normal = 0
	MT_Error  = 1
	MT_Ack    = 2
	MT_Delete = 3
	MT_Init   = 4
)

type elevatorMapMessage struct {
	Id          string
	ElevatorMap map[string]Elevator
	Message     MessageType
	Counter     int
}

// We make channels for sending and receiving our custom data types
var sendToNetwork = make(chan elevatorMapMessage)
var receivedFromNetwork = make(chan elevatorMapMessage)
var sendToLocalElevator = make(chan elevatorMapMessage, 10)

func network_sendElevatorMapMessage(elevators map[string]Elevator, message MessageType) {
	counter++
	fmt.Println("send", message, counter)

	elevatorMapMessage := elevatorMapMessage{LocalID, elevators, message, counter}

	if SingleElevatorMode {
		sendToLocalElevator <- elevatorMapMessage
	} else {
		sendToNetwork <- elevatorMapMessage
	}

}

func network_handler() {

	
	peerUpdateCh := make(chan peers.PeerUpdate)
	
	peerTxEnable := make(chan bool)
	go peers.Transmitter(42065, LocalID, peerTxEnable)
	go peers.Receiver(42065, peerUpdateCh)

	
	go bcast.Transmitter(42069, sendToNetwork)
	go bcast.Receiver(42069, receivedFromNetwork)

	fmt.Println("Started")
	for {
		select {
		case p := <-peerUpdateCh:

			if len(p.Peers) == 0 {
				SingleElevatorMode = true
			} else {
				SingleElevatorMode = false
			}
			fmt.Println("single: ", SingleElevatorMode)

			if len(p.Lost) > 0 {
				fmt.Println("Lost : ", p.Lost)
				for _, id := range p.Lost {
					if id != LocalID {

						lostElevator := ActiveElevatorMap[id]
						lostElevator.Error = true
						ActiveElevatorMap[id] = lostElevator
						network_sendElevatorMapMessage(ActiveElevatorMap, MT_Error)
					}

				}
			}

			if len(p.New) > 0 {
				fmt.Println("New: ", p.New)
				newElevator := ActiveElevatorMap[p.New]
				newElevator.Error = false
				ActiveElevatorMap[p.New] = newElevator
				network_sendElevatorMapMessage(ActiveElevatorMap, MT_Normal)
			}

			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

		case a := <-receivedFromNetwork:
			//
			fmt.Println("Incoming from network", a.Id, a.Counter)
			elevatorMap_handleIncomingMessage(a)

			//fmt.Printf("Received: %#v\n", a)

		case a := <-sendToLocalElevator:
			fmt.Println("sendToSelf")
			elevatorMap_handleIncomingMessage(a)
		}
	}
}

