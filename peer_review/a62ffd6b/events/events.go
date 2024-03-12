package events

import (
	"Project/Driver-go/elevio"
	"Project/order"
	"fmt"
	"time"
)

const (
	Idle = iota
	Moving
	DoorOpen
)

var ElevatorDir elevio.MotorDirection = elevio.MD_Up
var ElevatorLastDir elevio.MotorDirection
var ElevatorState int = Idle

var DoorTimer = time.NewTimer(3 * time.Second)

var ObSwitch = make(chan bool, 10)
var AtFloor = make(chan bool, 2)

func InitEvents() {
	DoorTimer.Stop()
}

func EventHandler(buttons chan elevio.ButtonEvent, floors chan int, obstr, stop chan bool, button *elevio.ButtonEvent, floor *int, obstructionSwitch, stopSwitch *bool) {
	for {
		select {
		case *button = <-buttons:
			fmt.Printf("%+v\n", *button)
			if *floor == (*button).Floor && ElevatorState != Moving {
				if ElevatorState == DoorOpen {
					ResetDoorTimer()
				} else if ElevatorState == Idle {
					ElevatorLastDir = ElevatorDir
					ElevatorDir = elevio.MD_Stop
				}
			} else {
				order.AddOrder(button)
			}
		case *floor = <-floors:
			AtFloor <- true
			elevio.SetFloorIndicator(*floor)
		case *obstructionSwitch = <-obstr:
			fmt.Printf("Obstruction channel: %+v\n", *obstructionSwitch)
			ObSwitch <- *obstructionSwitch
		case *stopSwitch = <-stop:
			fmt.Printf("Stop Switch: %+v\n", *stopSwitch)
		}
	}
}

func ResetDoorTimer() {
	DoorTimer.Reset(3 * time.Second)
	elevio.SetDoorOpenLamp(true)
}

func OpenDoor(floor *int) {
	ResetDoorTimer()
	order.ClearOrder(floor, ElevatorDir)
}

func CloseDoor(toIdleState chan bool) {
	for {
		select {
		case a := <-ObSwitch:
			if a {
				DoorTimer.Stop()
			} else {
				DoorTimer.Reset(3 * time.Second)
			}
		case <-DoorTimer.C:
			elevio.SetDoorOpenLamp(false)
			toIdleState <- true
			return
		}
	}
}
