package main

import (
	"Driver-go/elevio"
	"Network-go/network/localip"
	"flag"
	"fmt"
	"time"
)

const DOOR_OPEN_TIME time.Duration = 3 * time.Second
const FLOOR_ERROR_TIME time.Duration = 4 * time.Second
const DOOR_ERROR_TIME time.Duration = 8 * time.Second

var DoorOpenTimer = time.NewTimer(DOOR_OPEN_TIME)
var DoorErrorTimer = time.NewTimer(DOOR_ERROR_TIME)
var FloorErrorTimer = time.NewTimer(FLOOR_ERROR_TIME)

func main() {
	DoorOpenTimer.Stop()
	DoorErrorTimer.Stop()
	FloorErrorTimer.Stop()

	var id string
	var elevatorPort string

	flag.StringVar(&id, "id", "", "id of this peer")
	flag.StringVar(&elevatorPort, "port", "", "port of elevator")
	flag.Parse()

	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("heis-%s", localIP)
	}

	LocalID = id

	if elevatorPort == "" {
		elevatorPort = "15657"
	}

	elevio.Init("localhost:"+elevatorPort, N_FLOORS)

	var obstruction bool = false

	drv_buttons := make(chan elevio.ButtonEvent, 10)
	drv_floors := make(chan int, 10)
	drv_obstr := make(chan bool, 10)
	drv_stop := make(chan bool, 10)

	go network_handler()
	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	elevatorMap_init()

	for {
		select {
		case a := <-drv_buttons:
			fsm_onRequestButtonPress(int(a.Floor), elevio.ButtonType(a.Button))
		case a := <-drv_floors:
			fmt.Println("On floor Arrival", a)
			fsm_onFloorArrival(a)

		case a := <-drv_obstr:
			if a {
				obstruction = true
			} else {
				obstruction = false
			}

		case a := <-DoorOpenTimer.C:
			if obstruction {
				DoorOpenTimer.Reset(DOOR_OPEN_TIME)
			} else {
				fmt.Printf("%+v\n", a)

				LocalElevator.Error = false
				FloorErrorTimer.Stop()

				fsm_onDoorTimeout()
			}
		case <-FloorErrorTimer.C:
			LocalElevator.Orders = ActiveElevatorMap[LocalID].Orders

			fmt.Println("Floor timer")
			if !LocalElevator.Error {
				fmt.Println("FLOOR TIMEOUT")
				LocalElevator.Error = true
				ActiveElevatorMap[LocalID] = LocalElevator
				network_sendElevatorMapMessage(ActiveElevatorMap, MT_Error)
			}

			fmt.Println("Reset timer")
			FloorErrorTimer.Reset(1 * time.Second)

			if LocalElevator.Dirn == elevio.MD_Stop {
				LocalElevator.Dirn = elevio.MD_Down
				LocalElevator.Behaviour = EB_Moving
			}
			elevio.SetMotorDirection(LocalElevator.Dirn)
			ActiveElevatorMap[LocalID] = LocalElevator

		case <-DoorErrorTimer.C:
			fmt.Println("DOOR TIMEOUT")
			LocalElevator.Error = true
			ActiveElevatorMap[LocalID] = LocalElevator
			network_sendElevatorMapMessage(ActiveElevatorMap, MT_Error)
		}
	}
}
