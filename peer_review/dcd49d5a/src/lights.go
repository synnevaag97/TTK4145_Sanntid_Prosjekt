package main

import "Driver-go/elevio"

func lights_setAllHallLights(activeHallLights [N_FLOORS][N_BUTTONS]bool) {
	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS-1; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, activeHallLights[floor][btn])
		}
	}
}

func lights_setAllCabLights(es Elevator) {
	for floor := 0; floor < N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, es.Orders[floor][elevio.BT_Cab])
	}
}
