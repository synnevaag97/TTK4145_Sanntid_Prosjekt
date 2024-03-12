package main

import (
	"Driver-go/elevio"
	//"fmt"
)

type Action struct {
	Dirn      elevio.Dirn
	Behaviour ElevatorBehaviour
}

type Request struct {
	Floor   int
	BtnType int
}

func requests_areAbove(e Elevator) bool {
	for f := e.Floor + 1; f < elevio.N_FLOORS; f++ {
		for btn := 0; btn < elevio.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requests_areBelow(e Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < elevio.N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requests_areHere(e Elevator) bool {
	for btn := 0; btn < elevio.N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func requests_nextAction(e Elevator) Action {
	switch e.Dirn {
	case elevio.D_Up:
		if requests_areAbove(e) {
			return Action{elevio.D_Up, EB_Moving}
		}
		if requests_areHere(e) {
			return Action{elevio.D_Down, EB_DoorOpen}
		}
		if requests_areBelow(e) {
			return Action{elevio.D_Down, EB_Moving}
		}
		return Action{elevio.D_Stop, EB_Idle}
	case elevio.D_Down:
		if requests_areBelow(e) {
			return Action{elevio.D_Down, EB_Moving}
		}
		if requests_areHere(e) {
			return Action{elevio.D_Down, EB_DoorOpen}
		}
		if requests_areAbove(e) {
			return Action{elevio.D_Up, EB_Moving}
		}
		return Action{elevio.D_Stop, EB_Idle}
	case elevio.D_Stop:
		if requests_areHere(e) {
			return Action{elevio.D_Down, EB_DoorOpen}
		}
		if requests_areAbove(e) {
			return Action{elevio.D_Up, EB_Moving}
		}
		if requests_areBelow(e) {
			return Action{elevio.D_Down, EB_Moving}
		}
		return Action{elevio.D_Stop, EB_Idle}
	default:
		return Action{elevio.D_Stop, EB_Idle}
	}
}

func requests_shouldStop(e Elevator) bool {
	switch e.Dirn {
	case elevio.D_Down:
		return e.Requests[e.Floor][elevio.B_HallDown] || e.Requests[e.Floor][elevio.B_Cab] || !requests_areBelow(e)
	case elevio.D_Up:
		return e.Requests[e.Floor][elevio.B_HallUp] || e.Requests[e.Floor][elevio.B_Cab] || !requests_areAbove(e)
	case elevio.D_Stop:
		return true
	default:
		return false
	}
}

func requests_shouldClearImmediately(e Elevator, btn_floor int, btn_type elevio.Button) bool {
	switch e.Config.clearRequestVariant {
	case CV_All:
		return e.Floor == btn_floor
	case CV_InDirn:
		return e.Floor == btn_floor &&
			((e.Dirn == elevio.D_Up && btn_type == elevio.B_HallUp) ||
				(e.Dirn == elevio.D_Down && btn_type == elevio.B_HallDown) ||
				e.Dirn == elevio.D_Stop ||
				btn_type == elevio.B_Cab)

	default:
		return false
	}
}

func requests_clearAtCurrentFloor(e Elevator) (Elevator, []Request) {

	var hallRequests_copy[elevio.N_BUTTONS]bool
	for btn := 0; btn < elevio.N_BUTTONS-1; btn++ {
		hallRequests_copy[btn] = e.Requests[e.Floor][btn]
	}

	switch e.Config.clearRequestVariant {
	case CV_All:
		for btn := 0; btn < elevio.N_BUTTONS; btn++ {
			e.Requests[e.Floor][btn] = false
		}

	case CV_InDirn:
		e.Requests[e.Floor][elevio.B_Cab] = false
		switch e.Dirn {
		case elevio.D_Up:
			if !requests_areAbove(e) && !e.Requests[e.Floor][elevio.B_HallUp] {
				e.Requests[e.Floor][elevio.B_HallDown] = false
			}
			e.Requests[e.Floor][elevio.B_HallUp] = false

		case elevio.D_Down:
			if !requests_areBelow(e) && !e.Requests[e.Floor][elevio.B_HallDown] {
				e.Requests[e.Floor][elevio.B_HallUp] = false
			}
			e.Requests[e.Floor][elevio.B_HallDown] = false

		case elevio.D_Stop:
		default:
			e.Requests[e.Floor][elevio.B_HallUp] = false
			e.Requests[e.Floor][elevio.B_HallDown] = false

		}

	default:
		break
	}

	var clearedHallRequests []Request

	for btn := 0; btn < elevio.N_BUTTONS-1; btn++ {
		if e.Requests[e.Floor][btn] != hallRequests_copy[btn] {
			clearedHallRequests = append(clearedHallRequests, Request{e.Floor, btn})
		}
	}

	return e, clearedHallRequests
}
