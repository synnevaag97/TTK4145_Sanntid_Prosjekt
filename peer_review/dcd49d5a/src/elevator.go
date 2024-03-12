package main

import (
	"Driver-go/elevio"
	"fmt"
	"strconv"
)

const N_FLOORS = 4
const N_BUTTONS = 3

type ElevatorBehaviour int

const (
	EB_Idle     = 0
	EB_DoorOpen = 1
	EB_Moving   = 2
)

type Elevator struct {
	Floor     int
	Dirn      elevio.MotorDirection
	Orders  [N_FLOORS][N_BUTTONS]bool
	Behaviour ElevatorBehaviour
	Id        string
	Error     bool
}

func elevator_ebToString(eb ElevatorBehaviour) string {
	switch eb {
	case EB_Idle:
		return "EB_Idle"
	case EB_DoorOpen:
		return "EB_DoorOpen"
	case EB_Moving:
		return "EB_Moving"
	default:
		return "EB_Undefined"
	}
}

func elevator_dirnToString(md elevio.MotorDirection) string {
	switch md {
	case elevio.MD_Up:
		return "MD_Up"
	case elevio.MD_Down:
		return "MD_Down"
	case elevio.MD_Stop:
		return "MD_Stop"
	default:
		return "MD_Undefined"
	}
}


func elevator_print(es Elevator) {
	fmt.Println("  +--------------------+")
	fmt.Printf("  |floor = %-2d          |\n", es.Floor)
	fmt.Printf("  |dirn  = %-12.12s|\n", elevator_dirnToString(es.Dirn))
	fmt.Printf("  |behav = %-12.12s|\n", elevator_ebToString(es.Behaviour))
	fmt.Printf("  |error = %-12.12s|\n", strconv.FormatBool(es.Error))

	fmt.Println("  +--------------------+")
	fmt.Println("  |  | up  | dn  | cab |")
	for f := N_FLOORS - 1; f >= 0; f-- {
		fmt.Printf("  | %d", f)
		for btn := 0; btn < N_BUTTONS; btn++ {
			if (f == N_FLOORS-1 && elevio.ButtonType(btn) == elevio.BT_HallUp) || (f == 0 && elevio.ButtonType(btn) == elevio.BT_HallDown) {
				fmt.Printf("|     ")
			} else {
				if es.Orders[f][btn] {
					fmt.Printf("|  #  ")
				} else {
					fmt.Printf("|  -  ")
				}
			}
		}
		fmt.Printf("|\n")
	}
	fmt.Printf("  +--------------------+\n")
}

func elevators_print(elevs map[string]Elevator) {
	fmt.Printf("  +---------------------------------+\n")
	for id, element := range elevs {
		fmt.Println("ID: ", id)
		elevator_print(element)
	}
	fmt.Printf("  +---------------------------------+\n")

}
