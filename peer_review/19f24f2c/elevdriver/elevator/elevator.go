package elevator

import (
	"Elevdriver/config"
	"Elevdriver/data_structure"
	"Elevdriver/elevio"
	"fmt"
	"time"
)

const ( //States
	_                     = iota
	find_floor_s          = iota
	no_orders_s           = iota
	idle_s                = iota
	move_up_s             = iota
	move_down_s           = iota
	open_door_s           = iota
	wait_for_door_close_s = iota
)

const ( //Events
	_                        = iota
	floor_sensor_destination = iota
	floor_sensor             = iota
	obstr                    = iota
	no_orders                = iota
	moving_up                = iota
	moving_down              = iota
	received_new_queue_down  = iota
	received_new_queue_up    = iota
	received_new_queue_same  = iota
	motor_timeout            = iota
	unexpected_direction     = iota
	double_time              = iota
)

func calculate_state(event int, state int) int {
	switch state {
	case find_floor_s:
		switch event {
		case floor_sensor:
			return open_door_s
		case floor_sensor_destination:
			return open_door_s
		case motor_timeout:
			return find_floor_s
		default:
			return find_floor_s
		}

	case no_orders_s:
		switch event {
		case received_new_queue_same:
			return open_door_s
		case received_new_queue_down:
			return move_down_s
		case received_new_queue_up:
			return move_up_s
		case motor_timeout:
			return find_floor_s
		default:
			return no_orders_s
		}

	case move_up_s:
		switch event {
		case floor_sensor_destination:
			return open_door_s
		case motor_timeout:
			return find_floor_s
		default:
			return move_up_s
		}

	case move_down_s:
		switch event {
		case floor_sensor_destination:
			return open_door_s
		case motor_timeout:
			return find_floor_s
		default:
			return move_down_s
		}

	case open_door_s:
		fmt.Println("In open door state, event: ", event)
		switch event {
		case moving_up:
			return move_up_s
		case moving_down:
			return move_down_s
		case no_orders:
			return no_orders_s
		case floor_sensor_destination:
			return open_door_s
		case motor_timeout:
			return find_floor_s
		case unexpected_direction:
			return open_door_s
		case double_time:
			return open_door_s
		default:
			return wait_for_door_close_s
		}
	case wait_for_door_close_s:
		fmt.Println("In wait for door close state, event: ", event)
		switch event {
		case moving_up:
			return move_up_s
		case moving_down:
			return move_down_s
		case no_orders:
			return no_orders_s
		case floor_sensor_destination:
			return open_door_s
		case unexpected_direction:
			return open_door_s
		case double_time:
			return open_door_s
		case motor_timeout:
			return find_floor_s
		default:
			return wait_for_door_close_s
		}
	default:
		return find_floor_s
	}
}

func act_on_state(door_control_chan chan bool, state int, elevator_data data_structure.Elevator_data_t) data_structure.Elevator_data_t {
	switch state {
	case find_floor_s:
		elevio.SetMotorDirection(elevio.MD_Down)
		elevio.SetDoorOpenLamp(false)
		return update_elevator_data("moving", -1, "down")

	case no_orders_s:
		elevio.SetMotorDirection(elevio.MD_Stop)
		elevio.SetDoorOpenLamp(false)
		return update_elevator_data("idle", elevator_data.Floor, "stop")

	case move_up_s:
		elevio.SetMotorDirection(elevio.MD_Up)
		elevio.SetFloorIndicator(elevator_data.Floor)
		elevio.SetDoorOpenLamp(false)
		return update_elevator_data("moving", elevator_data.Floor, "up")

	case move_down_s:
		elevio.SetMotorDirection(elevio.MD_Down)
		elevio.SetFloorIndicator(elevator_data.Floor)
		elevio.SetDoorOpenLamp(false)
		return update_elevator_data("moving", elevator_data.Floor, "down")

	case open_door_s:
		elevio.SetMotorDirection(elevio.MD_Stop)
		elevio.SetFloorIndicator(elevator_data.Floor)
		elevio.SetDoorOpenLamp(true)
		go door_control(door_control_chan, 3*time.Second)
		return update_elevator_data("doorOpen", elevator_data.Floor, elevator_data.Direction)

	case wait_for_door_close_s:
		return elevator_data

	default:
		elevio.SetMotorDirection(elevio.MD_Stop)
		elevio.SetDoorOpenLamp(true)
		return update_elevator_data("idle", -1, "stop")
	}
}

func update_elevator_data(behaviour string, floor int, direction string) data_structure.Elevator_data_t {
	var elevator_data data_structure.Elevator_data_t
	elevator_data.Behaviour = behaviour
	elevator_data.Floor = floor
	elevator_data.Direction = direction
	return elevator_data
}

func Driver(
	elevator_data_channel chan data_structure.Elevator_data_t,
	arrived_at_floor_channel chan data_structure.Order_list_t,
	order_queue_chan chan [config.ORDER_QUEUE_SIZE]data_structure.Order_list_t,
	elevator_stuck_chan chan bool) {

	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	door_control_chan := make(chan bool)
	var elevator_data data_structure.Elevator_data_t
	var door_obstructed bool = false
	var found_floor bool
	var next_expected_direction int

	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)

	var order_queue [config.ORDER_QUEUE_SIZE]data_structure.Order_list_t

	state := calculate_state(0, 0)
	elevator_data = act_on_state(door_control_chan, state, elevator_data)
	time_out_timer := time.NewTimer(10 * time.Second) //Motor stuck timer

	elevator_data_channel <- elevator_data

	for i := range order_queue {
		order_queue[i].Floor = -1
	}

	for {
		fmt.Println("State: ", state)
		select {
		case a := <-drv_floors:
			fmt.Print("Found floor: ", a)
			elevator_stuck_chan <- false
			elevator_data.Floor = a
			found_floor = false
			var finished_order data_structure.Order_list_t
			for _, val := range order_queue {
				if val.Floor == a { //Check if call on floor is going in the right direction
					if val.Direction == elevio.BT_HallUp && (elevator_data.Direction == "up" || a == 0) {
						finished_order.Direction = val.Direction
						found_floor = true
						next_expected_direction = 1
					} else if val.Direction == elevio.BT_HallDown && (elevator_data.Direction == "down" || a == config.NUM_FLOORS-1) {
						finished_order.Direction = val.Direction
						found_floor = true
						next_expected_direction = -1
					} else if val.Direction == elevio.BT_Cab {
						finished_order.Direction = val.Direction
						found_floor = true
						next_expected_direction = 0
						finished_order.Floor = a
						fmt.Print("arrived_at_floor_channel ")
						arrived_at_floor_channel <- finished_order
						fmt.Println("GOOD")
					} else if !order_in_direction(order_queue, elevator_data.Floor, elevator_data.Direction) {
						fmt.Println(elevator_data)
						next_expected_direction = -next_expected_direction
						found_floor = true
						finished_order.Direction = val.Direction
					}
				}
			}

			if found_floor { //Stop at floor
				fmt.Println(" which is destination")
				time_out_timer.Stop()
				state = calculate_state(floor_sensor_destination, state)
				fmt.Println("Arrived at floor state: ", state)
				finished_order.Floor = a
				arrived_at_floor_channel <- finished_order
				for i, val := range order_queue {
					if val == finished_order || (val.Floor == a && val.Direction == elevio.BT_Cab) {
						order_queue[i].Floor = -1
					}
				}
				fmt.Println("Expected direction: ", next_expected_direction)
			} else if (a == 0 && elevator_data.Direction == "down") || (a == config.NUM_FLOORS-1 && elevator_data.Direction == "up") { //safety net
				fmt.Println("Elevator tried to go out of bounds at floor: ", a)
				state = calculate_state(floor_sensor_destination, state)
				elevator_data = act_on_state(door_control_chan, state, elevator_data)
				time_out_timer.Reset(config.MOTOR_TIMEOUT)
			} else { // Go past
				fmt.Println(".")
				state = calculate_state(floor_sensor, state)
				time_out_timer.Reset(config.MOTOR_TIMEOUT)
			}
			elevator_data = act_on_state(door_control_chan, state, elevator_data)

		case a := <-drv_obstr:
			door_obstructed = a
			if door_obstructed {
				fmt.Println("Obstruksjon aktiv")
			} else {
				fmt.Println("Obstruksjon inaktiv")
			}
			state = calculate_state(obstr, state)
			elevator_data = act_on_state(door_control_chan, state, elevator_data)

		case a := <-door_control_chan:
			fmt.Println("Door control")
			if door_obstructed && a { //If obstructed, wait for 500 ms, and check if still obstructed.
				go door_control(door_control_chan, 500*time.Millisecond)
			} else {
				order, button_type := get_order_from_queue(order_queue, elevator_data)
				switch order {
				case -1:
					if next_expected_direction != 1 || !order_exists(order_queue, elevator_data.Floor, elevio.BT_HallDown) {
						state = calculate_state(moving_down, state)
						time_out_timer.Reset(config.MOTOR_TIMEOUT)
					} else {
						fmt.Println("Turning around. Expected: ", next_expected_direction)
						next_expected_direction = -1
						state = calculate_state(unexpected_direction, state)
						arrived_at_floor_channel <- data_structure.Order_list_t{elevator_data.Floor, elevio.BT_HallDown}
					}
				case 0:
					state = calculate_state(double_time, state)
					finished_order := data_structure.Order_list_t{elevator_data.Floor, button_type}
					arrived_at_floor_channel <- finished_order
					for i, val := range order_queue {
						if val == finished_order {
							order_queue[i].Floor = -1
						}
					}

					time_out_timer.Stop()
				case 1:
					if next_expected_direction != -1 || !order_exists(order_queue, elevator_data.Floor, elevio.BT_HallUp) {
						state = calculate_state(moving_up, state)
						time_out_timer.Reset(config.MOTOR_TIMEOUT)
					} else {
						fmt.Println("Turning around. Expected: ", next_expected_direction)
						next_expected_direction = 1
						state = calculate_state(unexpected_direction, state)
						arrived_at_floor_channel <- data_structure.Order_list_t{elevator_data.Floor, elevio.BT_HallUp}
					}
				case 2:
					state = calculate_state(no_orders, state)
					time_out_timer.Stop()
				default:
					time_out_timer.Stop()
				}
				elevator_data = act_on_state(door_control_chan, state, elevator_data)
			}

		//Receive new queue from dist.
		case a := <-order_queue_chan:
			fmt.Println("Job queue rec.")
			order_queue = a
			direction, button_type := get_order_from_queue(order_queue, elevator_data)
			switch direction {
			case -1:
				state = calculate_state(received_new_queue_down, state)
				if elevator_data.Behaviour == "idle" {
					time_out_timer.Reset(config.MOTOR_TIMEOUT)
				}
			case 0:
				if state != open_door_s && state != wait_for_door_close_s {
					arrived_at_floor_channel <- data_structure.Order_list_t{elevator_data.Floor, button_type}
					time_out_timer.Stop()
				}
				state = calculate_state(received_new_queue_same, state)
			case 1:
				state = calculate_state(received_new_queue_up, state)
				if elevator_data.Behaviour == "idle" {
					time_out_timer.Reset(config.MOTOR_TIMEOUT)
				}
			default:
				time_out_timer.Stop()
			}
			elevator_data = act_on_state(door_control_chan, state, elevator_data)

		case <-time_out_timer.C:
			fmt.Println("Motor timeout")
			time_out_timer.Reset(1 * time.Second)
			state = calculate_state(motor_timeout, state)
			elevator_data = act_on_state(door_control_chan, state, elevator_data)
			elevator_stuck_chan <- true
		}
		fmt.Println("Elevator heartbeat")
		elevator_data_channel <- elevator_data
	}
}

func door_control(door_control_chan chan bool, time_till_shutdown time.Duration) {
	time.Sleep(time_till_shutdown)
	door_control_chan <- true
}

//Checks what the next reasonable order in the queue is. This depends on the direction the elevator is going.
func get_order_from_queue(order_queue [config.ORDER_QUEUE_SIZE]data_structure.Order_list_t, elevator_data data_structure.Elevator_data_t) (int, elevio.ButtonType) { //-1 = down, 0 = same, 1 = up 2 = no orders
	temp_direction := 2
	var temp_button_type elevio.ButtonType = elevio.BT_Cab
	for _, a := range order_queue {
		if a.Floor > -1 && a.Floor < config.NUM_FLOORS {
			switch {
			case elevator_data.Direction == "down":
				if a.Floor < elevator_data.Floor {
					return -1, a.Direction
				} else if elevator_data.Floor == a.Floor {
					temp_direction = 0
					temp_button_type = a.Direction
				} else {
					temp_direction = 1
					temp_button_type = a.Direction
				}
			case elevator_data.Direction == "up":
				if a.Floor > elevator_data.Floor {
					return 1, a.Direction
				} else if elevator_data.Floor == a.Floor {
					temp_direction = 0
					temp_button_type = a.Direction
				} else {
					temp_direction = -1
					temp_button_type = a.Direction
				}
			case elevator_data.Direction == "stop":
				if a.Floor > elevator_data.Floor {
					return 1, a.Direction
				} else if a.Floor < elevator_data.Floor {
					return -1, a.Direction
				} else {
					return 0, a.Direction
				}
			default:
				return 0, elevio.BT_Cab
			}
		}
	}
	return temp_direction, temp_button_type
}

func order_exists(order_queue [config.ORDER_QUEUE_SIZE]data_structure.Order_list_t, floor int, direction elevio.ButtonType) bool {
	for _, order := range order_queue {
		if order.Floor == floor && order.Direction == direction {
			return true
		}
	}
	return false
}

func order_in_direction(queue [config.ORDER_QUEUE_SIZE]data_structure.Order_list_t, floor int, dir string) bool {
	for _, order := range queue {
		if order.Floor != -1 {
			if (dir == "up" && order.Floor < floor) || (dir == "down" && order.Floor > floor) {
				return true
			}
		}
	}
	return false
}
