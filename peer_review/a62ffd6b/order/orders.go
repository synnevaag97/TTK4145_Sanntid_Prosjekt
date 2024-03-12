package order

import (
	"Project/Driver-go/elevio"
	"time"
)

const (
	_pollRate      = 100 * time.Millisecond
	NumFloors      = 4
	NumButtonTypes = 3
)

var ElevatorOrders [NumFloors][NumButtonTypes]int
var ElevatorRequest [NumFloors][NumButtonTypes]int
var ElevatorCompleted [NumFloors][NumButtonTypes]int

func AddOrder(button *elevio.ButtonEvent) {
	if button.Button == elevio.BT_Cab {
		ElevatorOrders[(*button).Floor][button.Button] = 1
	} else {
		ElevatorRequest[(*button).Floor][button.Button] = 1
	}
}

func ClearOrder(floor *int, direction elevio.MotorDirection) {
	elevio.SetButtonLamp(elevio.BT_Cab, *floor, false)
	if direction == elevio.MD_Up && ElevatorOrders[*floor][elevio.BT_HallUp] == 1 {
		ElevatorOrders[*floor][elevio.BT_HallUp] = 0
		ElevatorCompleted[*floor][elevio.BT_HallUp] = 1
	} else if direction == elevio.MD_Down && ElevatorOrders[*floor][elevio.BT_HallDown] == 1 {
		ElevatorOrders[*floor][elevio.BT_HallDown] = 0
		ElevatorCompleted[*floor][elevio.BT_HallDown] = 1
	}
	ElevatorOrders[*floor][elevio.BT_Cab] = 0
}

func PendingOrders(min, max int) bool {
	sum := 0
	for i := min; i < max; i++ {
		for j := 0; j < NumButtonTypes; j++ {
			if ElevatorOrders[i][j] == 1 {
				sum += 1
			}
		}
	}
	if sum > 0 {
		return true
	} else {
		return false
	}
}

func UpdateCabOrderLights() {
	var j elevio.ButtonType
	for i := 0; i < NumFloors; i++ {
		for j = NumButtonTypes - 1; j < NumButtonTypes; j++ {
			elevio.SetButtonLamp(j, i, elevio.IntToBool(ElevatorOrders[i][j]))
		}
	}
}
