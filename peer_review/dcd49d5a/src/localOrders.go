package main

import (
	"Driver-go/elevio"
)


type Action struct {
	Dirn      elevio.MotorDirection
	Behaviour ElevatorBehaviour
}

func localOrders_above(e Elevator) bool {
	for floor := e.Floor + 1; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Orders[floor][btn] {
				return true
			}
		}
	}
	return false
}

func localOrders_below(e Elevator) bool {
	for floor := 0; floor < e.Floor; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Orders[floor][btn] {
				return true
			}
		}
	}
	return false
}

func localOrders_here(e Elevator) bool {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.Orders[e.Floor][btn] {
			return true
		}
	}
	return false
}

func localOrders_nextAction(e Elevator) Action {

	switch dirn := e.Dirn; dirn {
	case elevio.MD_Up:
		if localOrders_above(e) {
			return Action{elevio.MD_Up, EB_Moving}
		} else if localOrders_here(e) {
			return Action{elevio.MD_Down, EB_DoorOpen}
		} else if localOrders_below(e) {
			return Action{elevio.MD_Down, EB_Moving}
		} else if !localOrders_below(e) {
			return Action{elevio.MD_Stop, EB_Idle}
		}
	case elevio.MD_Down:
		if localOrders_below(e) {
			return Action{elevio.MD_Down, EB_Moving}
		} else if localOrders_here(e) {
			return Action{elevio.MD_Up, EB_DoorOpen}
		} else if localOrders_above(e) {
			return Action{elevio.MD_Up, EB_Moving}
		} else if !localOrders_above(e) {
			return Action{elevio.MD_Stop, EB_Idle}
		}
	case elevio.MD_Stop:
		if localOrders_here(e) {
			return Action{elevio.MD_Stop, EB_DoorOpen}
		} else if localOrders_above(e) {
			return Action{elevio.MD_Up, EB_Moving}
		} else if localOrders_below(e) {
			return Action{elevio.MD_Down, EB_Moving}
		} else if !localOrders_below(e) {
			return Action{elevio.MD_Stop, EB_Idle}
		}
	default:
		return Action{elevio.MD_Stop, EB_Idle}
	}
	return Action{elevio.MD_Stop, EB_Idle} 
}

func localOrders_shouldStop(e Elevator) bool {
	switch dirn := e.Dirn; dirn {

	case elevio.MD_Down:
		return e.Orders[e.Floor][elevio.BT_HallDown] || e.Orders[e.Floor][elevio.BT_Cab] || !localOrders_below(e)

	case elevio.MD_Up:
		return e.Orders[e.Floor][elevio.BT_HallUp] || e.Orders[e.Floor][elevio.BT_Cab] || !localOrders_above(e)
	default:
		return true
	}

}

func localOrders_shouldClearImmediately(e Elevator, btn_floor int, btn_type elevio.ButtonType) bool {
	
	return e.Floor == btn_floor &&
		((e.Dirn == elevio.MD_Up && btn_type == elevio.BT_HallUp) ||
			(e.Dirn == elevio.MD_Down && btn_type == elevio.BT_HallDown) ||
			e.Dirn == elevio.MD_Stop ||
			btn_type == elevio.BT_Cab)

}

func localOrders_clearAtCurrentFloor(e Elevator) Elevator {
	e.Orders[e.Floor][elevio.BT_Cab] = false

	switch e.Dirn {
	case elevio.MD_Up:
		if !localOrders_above(e) && !e.Orders[e.Floor][elevio.BT_HallUp] {
			e.Orders[e.Floor][elevio.BT_HallDown] = false
		}
		e.Orders[e.Floor][elevio.BT_HallUp] = false

	case elevio.MD_Down:
		if !localOrders_below(e) && !e.Orders[e.Floor][elevio.BT_HallDown] {
			e.Orders[e.Floor][elevio.BT_HallUp] = false
		}
		e.Orders[e.Floor][elevio.BT_HallDown] = false
	case elevio.MD_Stop:
		e.Orders[e.Floor][elevio.BT_HallUp] = false
		e.Orders[e.Floor][elevio.BT_HallDown] = false
	default:
		e.Orders[e.Floor][elevio.BT_HallUp] = false
		e.Orders[e.Floor][elevio.BT_HallDown] = false
	}

	return e
}
