package singleElevator

import (
	"PROJECT-GROUP-[REDACTED]/config"
	"PROJECT-GROUP-[REDACTED]/elevio"
	"PROJECT-GROUP-[REDACTED]/networking"
	"encoding/json"
	"os"
)

type elevator_status struct {
	floor     int
	direction int
}

type floor_info struct {
	up   bool
	down bool
	cab  bool
}

var floor [config.NUMBER_OF_FLOORS]floor_info
var elevator elevator_status
var elevator_command elevator_status

func HallOrder(
	ch_new_order chan<- bool,
	ch_elevator_has_arrived chan<- bool,
	ch_command_elev <-chan elevio.ButtonEvent,
	ch_update_elevator_node_order chan<- update_elevator_node,
	ch_remove_elevator_node_order chan<- update_elevator_node,
	ch_req_ID chan int,
	ch_req_data chan networking.Elevator_node,
) {
	for {
		select {
		case a := <-ch_command_elev:
			tot_hall_calls := networking.UpdateHallCallsTot(ch_req_ID, ch_req_data)
			if tot_hall_calls[a.Floor].Up && a.Button == elevio.BT_HallUp || tot_hall_calls[a.Floor].Down && a.Button == elevio.BT_HallDown {
				//If order already exists somewhere, decline it
			} else if current_state == idle && a.Floor == elevator.floor {
				ch_elevator_has_arrived <- true //Elevator has arrived if elevator already standing still at correct floor
			} else {
				switch a.Button {
				case elevio.BT_HallUp:
					floor[a.Floor].up = true
					add_order_to_node.command = "update order up"
					add_order_to_node.update_value = a.Floor
					ch_update_elevator_node_order <- add_order_to_node
					elevio.SetButtonLamp(elevio.BT_HallUp, a.Floor, true)
				case elevio.BT_HallDown:
					floor[a.Floor].down = true
					add_order_to_node.command = "update order down"
					add_order_to_node.update_value = a.Floor
					ch_update_elevator_node_order <- add_order_to_node
					elevio.SetButtonLamp(elevio.BT_HallDown, a.Floor, true)
				case elevio.BT_Cab:
					floor[a.Floor].cab = true
					elevio.SetButtonLamp(elevio.BT_Cab, a.Floor, true)
					updateCabCallJson(true, a.Floor)
				}
				if current_state != moving {
					ch_new_order <- true
				}
			}
		}
	}
}

func RemoveOrder(level int, direction int, ch_remove_elevator_node_order chan<- update_elevator_node) {
	floor[level].cab = false
	elevio.SetButtonLamp(2, level, false)
	updateCabCallJson(false, level)
	if direction == int(elevio.MD_Up) {
		if !floor[level].up {
			floor[level].down = false
			sendRemoveOrder("remove order down", level, ch_remove_elevator_node_order)
			elevio.SetButtonLamp(elevio.BT_HallDown, level, false)
		} else {
			floor[level].up = false
			sendRemoveOrder("remove order up", level, ch_remove_elevator_node_order)
			elevio.SetButtonLamp(elevio.BT_HallUp, level, false)
		}
	} else if direction == int(elevio.MD_Down) {
		if !floor[level].down {
			floor[level].up = false
			sendRemoveOrder("remove order up", level, ch_remove_elevator_node_order)
			elevio.SetButtonLamp(elevio.BT_HallUp, level, false)
		} else {
			floor[level].down = false
			sendRemoveOrder("remove order down", level, ch_remove_elevator_node_order)
			elevio.SetButtonLamp(elevio.BT_HallDown, level, false)
		}
	} else if direction == int(elevio.MD_Stop) {
		if !floor[level].down {
			floor[level].up = false
			sendRemoveOrder("remove order up", level, ch_remove_elevator_node_order)
			elevio.SetButtonLamp(elevio.BT_HallUp, level, false)
		} else {
			floor[level].down = false
			sendRemoveOrder("remove order down", level, ch_remove_elevator_node_order)
			elevio.SetButtonLamp(elevio.BT_HallDown, level, false)
		}
	}
}

func sendRemoveOrder(command string, level int, ch_remove_elevator_node_order chan<- update_elevator_node) {
	remove_order_from_node.command = command
	remove_order_from_node.update_value = level
	ch_remove_elevator_node_order <- remove_order_from_node
}

func updateCabCallJson(command bool, floor int) {
	file, _ := os.OpenFile("cabcalls.json", os.O_RDWR|os.O_CREATE, 0666)
	cab_calls[floor] = command
	bytes, _ := json.Marshal(cab_calls)
	file.Truncate(0)
	file.WriteAt(bytes, 0)
	file.Close()
}

func requestAbove() bool {
	for i := elevator.floor + 1; i < config.NUMBER_OF_FLOORS; i++ {
		if floor[i].up || floor[i].cab {
			elevator_command.floor = i
			elevator_command.direction = int(elevio.MD_Up)
			return true
		}
	}
	for i := 3; i > elevator.floor; i-- {
		if floor[i].down {
			elevator_command.floor = i
			elevator_command.direction = int(elevio.MD_Up)
			return true
		}
	}
	return false
}

func requestHere() bool {
	if floor[elevator.floor].up || floor[elevator.floor].down || floor[elevator.floor].cab {
		elevator_command.floor = elevator.floor
		elevator_command.direction = int(elevio.MD_Stop)
		return true
	}
	return false
}

func requestBelow() bool {
	for i := elevator.floor - 1; i >= 0; i-- {
		if floor[i].down || floor[i].cab {
			elevator_command.floor = i
			elevator_command.direction = int(elevio.MD_Down)
			return true
		}
	}
	for i := 0; i < elevator.floor; i++ {
		if floor[i].up {
			elevator_command.floor = i
			elevator_command.direction = int(elevio.MD_Down)
			return true
		}
	}
	return false
}

func RequestNextAction(direction int) bool {
	switch direction {
	case int(elevio.MD_Up):
		if requestAbove() {
			return true
		} else if requestHere() {
			return true
		} else if requestBelow() {
			return true
		}

	case int(elevio.MD_Down):
		if requestBelow() {
			return true
		} else if requestHere() {
			return true
		} else if requestAbove() {
			return true
		}

	case int(elevio.MD_Stop):
		if requestAbove() {
			return true
		} else if requestHere() {
			return true
		} else if requestBelow() {
			return true
		}
	}
	return false
}

func UpdatePosition(level int, direction int, ch_remove_elevator_node_order chan<- update_elevator_node) {
	elevator.floor = level
	elevator.direction = direction
	RemoveOrder(level, direction, ch_remove_elevator_node_order)
}
