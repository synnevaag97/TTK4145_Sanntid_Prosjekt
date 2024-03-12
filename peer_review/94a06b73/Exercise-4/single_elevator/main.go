package main

import (
	"Elevator-go/elevator"
	"Elevator-go/elevio"
)

func main() {
	elevio.Init("localhost:48613", 4)

	ch_onRequestButtonPress := make(chan elevio.ButtonEvent)
	ch_onFloorArrival := make(chan int)
	ch_obstruction := make(chan bool, 1)
	//ch_timerDoor := make(chan bool)

	go elevio.PollFloorSensor(ch_onFloorArrival)
	go elevio.PollObstructionSwitch(ch_obstruction)
	go elevio.PollButtons(ch_onRequestButtonPress)

	elevator.Elevator(
		ch_onRequestButtonPress,
		ch_onFloorArrival,
		ch_obstruction)
	//ch_timerDoor)

	select {}
}
