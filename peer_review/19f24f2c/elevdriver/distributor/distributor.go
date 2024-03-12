package distributor

import (
	"Elevdriver/backup"
	"Elevdriver/config"
	"Elevdriver/data_structure"
	"Elevdriver/elevator"
	"Elevdriver/elevio"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

func Distributor(elevator_id int,
	systemInfo data_structure.System_info_t,
	arbitration_status_receive_chan chan data_structure.Arbitration_t,
	stuck_send_chan chan bool,
	elevator_data_receive_channel chan data_structure.Received_elevator_data_t,
	elevator_data_send_channel chan data_structure.Elevator_data_t,
	order_send_chan chan data_structure.Order_t,
	order_receive_chan chan data_structure.Order_t,
	order_queue_send_chan chan [config.ORDER_QUEUE_SIZE]data_structure.Order_t,
	order_queue_receive_chan chan [config.ORDER_QUEUE_SIZE]data_structure.Order_t,
	hall_request_send_chan chan [config.NUM_ELEVATORS][config.NUM_FLOORS][2]bool,
	hall_request_receive_chan chan [config.NUM_FLOORS][2]bool,
	cab_request_to_backup chan data_structure.Elevator_data_t) {

	var arbitration_status data_structure.Arbitration_t
	var order_queue [config.ORDER_QUEUE_SIZE]data_structure.Order_t
	var cost_data data_structure.Cost_data_t
	cost_data.States = make(map[int]*data_structure.Elevator_data_t, config.NUM_ELEVATORS)
	var latest_hall_requests [config.NUM_FLOORS][2]bool
	var temp_hall_requests [config.NUM_ELEVATORS][config.NUM_FLOORS][2]bool
	var requests_changed bool
	var order_queue_changed bool

	drv_buttons := make(chan elevio.ButtonEvent)
	job_queue_chan := make(chan [config.ORDER_QUEUE_SIZE]data_structure.Order_list_t, 2)
	elevator_data_channel := make(chan data_structure.Elevator_data_t, 2)
	arrrived_at_floor_channel := make(chan data_structure.Order_list_t)
	elevator_stuck_channel := make(chan bool, 2)

	order_ticker := time.NewTicker(config.SEND_INTERVAL)
	elevator_data_ticker := time.NewTicker(1 * time.Second)
	refresh_queue_ticker := time.NewTicker(5 * time.Second)

	go elevio.PollButtons(drv_buttons)
	go elevator.Driver(elevator_data_channel, arrrived_at_floor_channel, job_queue_chan, elevator_stuck_channel)

	//Get CabReq. backup
	elevator_data, err := backup.GetBackup(systemInfo)
	if err != nil {
		fmt.Println("Error reading backup.")
		for v := range elevator_data.CabRequests {
			elevator_data.CabRequests[v] = false
		}
		elevator_data = update_elevator_data("idle", -1, "stop", elevator_data)
	}

	//Reset lights
	for i := 0; i < 2; i++ {
		for j := 0; j < config.NUM_FLOORS; j++ {
			elevio.SetButtonLamp(elevio.ButtonType(i), j, false)
		}
	}
	for i, light := range elevator_data.CabRequests {
		elevio.SetButtonLamp(elevio.BT_Cab, i, light)
	}

	fmt.Println(elevator_data)
	job_queue_chan <- create_order_list(elevator_data.CabRequests, latest_hall_requests)

	//Clear order queue
	for i := range order_queue {
		order_queue[i].Finished = true
	}
	order_queue_changed = true
	start_time := time.Now().UnixMilli()

	for {
		fmt.Println("Time since start: ", time.Now().UnixMilli()-start_time, "ms, Master: ", arbitration_status.Is_master, "Connected: ", arbitration_status.Connected)
		select {

		case a := <-arbitration_status_receive_chan:
			arbitration_status = a

		case a := <-drv_buttons:
			fmt.Println("ButtonEvent: ", a.Button, " Floor: ", a.Floor)
			//CabReq. is just saved locally
			if a.Button == elevio.BT_Cab {
				elevator_data.CabRequests[a.Floor] = true
				cab_request_to_backup <- elevator_data
				if arbitration_status.Is_master {
					cost_data.States[elevator_id] = &elevator_data
				} else {
					elevator_data_send_channel <- elevator_data
				}
				job_queue_chan <- create_order_list(elevator_data.CabRequests, latest_hall_requests)
				//Doesn't take hall calls unlesss connected to at least 1 other elevator.
			} else if arbitration_status.Connected {
				new_order := data_structure.Order_t{a.Floor, a.Button, false}
				if !order_already_in_queue(new_order, order_queue) {
					//If master: distribute orders, If slave: send order to master.
					if arbitration_status.Is_master {
						//Find free space in order_queue for the new order
						for i, order := range order_queue {
							if order.Finished {
								order_queue[i] = new_order
								break
							}
						}
						order_queue_changed = true
						cost_data = recalculate_cost_data(order_queue, cost_data, arbitration_status.Alive_list)
						temp_hall_requests = assign_orders(cost_data)
						latest_hall_requests = temp_hall_requests[elevator_id]
						requests_changed = true
						elevio.SetButtonLamp(a.Button, a.Floor, true)
						job_queue_chan <- create_order_list(elevator_data.CabRequests, latest_hall_requests)
					} else {
						order_send_chan <- new_order
					}
				} else {
					fmt.Println("Order exists already")
				}
			}

		//A received order queue means that lights can be turned on/off according to it.
		case a := <-order_queue_receive_chan:
			var elevator_panel [config.NUM_FLOORS][2]bool
			for _, order := range a {
				if !order.Finished {
					elevator_panel[order.Floor][int(order.Direction)] = true
				}
			}

			for floor, a := range elevator_panel {
				for dir, val := range a {
					elevio.SetButtonLamp(elevio.ButtonType(dir), floor, val)
				}
			}
			order_queue = a

		case a := <-hall_request_receive_chan:
			latest_hall_requests = a
			job_queue_chan <- create_order_list(elevator_data.CabRequests, latest_hall_requests)

		//When elevator arrives at floor in queue, the light and order are cleared.
		//Slave sends order done to master.
		case a := <-arrrived_at_floor_channel:
			fmt.Println("Found floor receive.")
			elevio.SetButtonLamp(a.Direction, a.Floor, false)
			if a.Direction == elevio.BT_Cab {
				elevator_data.CabRequests[a.Floor] = false
				cab_request_to_backup <- elevator_data
				if arbitration_status.Is_master {
					cost_data.States[elevator_id] = &elevator_data
				} else {
					elevator_data_send_channel <- elevator_data
				}
			}
			if arbitration_status.Is_master || !arbitration_status.Connected {
				for i, order := range order_queue {
					if order.Floor == a.Floor && !order.Finished && (order.Direction == a.Direction || order.Direction == elevio.BT_Cab) {
						order_queue[i].Finished = true
						latest_hall_requests[a.Floor][a.Direction] = false
						cost_data.HallRequests = latest_hall_requests
						fmt.Println("Finished: ", order_queue[i])
						order_queue_changed = true
					}
				}
			} else {
				order_send_chan <- data_structure.Order_t{a.Floor, a.Direction, true}
				if a.Direction != elevio.BT_Cab {
					latest_hall_requests[a.Floor][int(a.Direction)] = false
				}
				job_queue_chan <- create_order_list(elevator_data.CabRequests, latest_hall_requests)
			}

		//Add/remove incoming order to order_queue, and distribute it to slaves
		case a := <-order_receive_chan:
			if a.Finished {
				for i, order := range order_queue {
					if !order.Finished && order.Floor == a.Floor && order.Direction == a.Direction {
						order_queue[i].Finished = true
						elevio.SetButtonLamp(a.Direction, a.Floor, false)
						cost_data.HallRequests[order.Floor][int(order.Direction)] = false
					}
				}
				order_queue_changed = true
			} else {
				for i, order := range order_queue {
					if order.Finished {
						order_queue[i] = a
						order_queue_changed = true
						elevio.SetButtonLamp(a.Direction, a.Floor, true)
						break
					}
				}
				order_queue_changed = true
				cost_data = recalculate_cost_data(order_queue, cost_data, arbitration_status.Alive_list)
				temp_hall_requests = assign_orders(cost_data)
				fmt.Println(temp_hall_requests)
				latest_hall_requests = temp_hall_requests[elevator_id]
				job_queue_chan <- create_order_list(elevator_data.CabRequests, latest_hall_requests)
				requests_changed = true
			}

		//Master received from slaves
		case a := <-elevator_data_receive_channel:
			cost_data.States[a.Elevator_id] = &a.Elevator_data

		//Received from elevator.go
		case a := <-elevator_data_channel:
			elevator_data.Behaviour = a.Behaviour
			elevator_data.Floor = a.Floor
			elevator_data.Direction = a.Direction
			cost_data.States[elevator_id] = &elevator_data
			if !arbitration_status.Is_master {
				elevator_data_send_channel <- elevator_data
			}

		case a := <-elevator_stuck_channel:
			if a {
				fmt.Println("Received elevator stuck.")
				stuck_send_chan <- false
				fmt.Println("GOOD")
			} else {
				fmt.Print("DIST: Sending stuck false ")
				stuck_send_chan <- true
				fmt.Println("GOOD")

			}

		//Ticker based sender. Relieves the elevator when two orders are recevied at once.
		case <-order_ticker.C:
			fmt.Println("Send order queue, and request.")
			if order_queue_changed && arbitration_status.Is_master {
				fmt.Println("Order_queue sent: ", order_queue)
				order_queue_changed = false
				order_queue_send_chan <- order_queue
			}

			if requests_changed && arbitration_status.Is_master {
				requests_changed = false
				hall_request_send_chan <- temp_hall_requests
			}

		case <-elevator_data_ticker.C:
			fmt.Println("Send elevator data")
			if !arbitration_status.Is_master {
				elevator_data_channel <- elevator_data
			}

		case <-refresh_queue_ticker.C:
			fmt.Println("Checking order queue for unfufilled orders: ", order_queue_empty(order_queue))
			if arbitration_status.Is_master || !arbitration_status.Connected {
				if !order_queue_empty(order_queue) {
					cost_data = recalculate_cost_data(order_queue, cost_data, arbitration_status.Alive_list)
					temp_hall_requests = assign_orders(cost_data)
					fmt.Println(temp_hall_requests)
					latest_hall_requests = temp_hall_requests[elevator_id]
					job_queue_chan <- create_order_list(elevator_data.CabRequests, latest_hall_requests)
					requests_changed = true
				}
			}

		case <-time.After(5 * time.Second):
			order_queue_changed = true
			fmt.Println("Halleluja amen")
		}
	}
}

func order_queue_empty(order_queue [config.ORDER_QUEUE_SIZE]data_structure.Order_t) bool {
	for _, order := range order_queue {
		if !order.Finished {
			return false
		}
	}
	return true
}

func order_already_in_queue(order data_structure.Order_t,
	order_queue [config.ORDER_QUEUE_SIZE]data_structure.Order_t) bool {
	for _, a := range order_queue {
		if a == order && a.Finished {
			return true
		}
	}
	return false
}

//Creates new cost_data map with all currently alive elevators.
func recalculate_cost_data(orders [config.ORDER_QUEUE_SIZE]data_structure.Order_t,
	old_cost_data data_structure.Cost_data_t,
	is_alive [config.NUM_ELEVATORS]bool) data_structure.Cost_data_t {

	var cost_data data_structure.Cost_data_t

	var num_alive int = 0

	for _, status := range is_alive {
		if !status {
			num_alive++
		}
	}

	cost_data.States = make(map[int]*data_structure.Elevator_data_t, num_alive)

	var state_iterator int = 0
	for _, status := range is_alive {
		if status {
			cost_data.States[state_iterator] = old_cost_data.States[state_iterator]
		}
		state_iterator++
	}

	for _, order := range orders {
		if !order.Finished {
			cost_data.HallRequests[order.Floor][order.Direction] = true
		}
	}
	return cost_data
}

func assign_orders(cost_data data_structure.Cost_data_t) [config.NUM_ELEVATORS][config.NUM_FLOORS][2]bool {
	ex, err := json.Marshal(cost_data)
	var Json_result []byte

	if runtime.GOOS == "windows" {
		Json_result, err = exec.Command("./hall_request_assigner.exe", "--input", string(ex)).Output()
	} else {
		Json_result, err = exec.Command("./hall_request_assigner", "--input", string(ex)).Output()
	}
	if err != nil {
		fmt.Println(err)
		return *new([config.NUM_ELEVATORS][config.NUM_FLOORS][2]bool)
	}
	Json_out := new(map[int][config.NUM_FLOORS][2]bool)
	err = json.Unmarshal(Json_result, &Json_out)
	if err != nil {
		fmt.Println(err)
		return *new([config.NUM_ELEVATORS][config.NUM_FLOORS][2]bool)
	}

	var assign_out [config.NUM_ELEVATORS][config.NUM_FLOORS][2]bool
	for id := range *Json_out {
		assign_out[id] = (*Json_out)[id]
	}

	return assign_out
}

//Creates a list off all jobs currently assigned to the elevator.
func create_order_list(Cab_requests [config.NUM_FLOORS]bool,
	delegated_hall_requests [config.NUM_FLOORS][2]bool) [config.ORDER_QUEUE_SIZE]data_structure.Order_list_t {

	var order_list [config.ORDER_QUEUE_SIZE]data_structure.Order_list_t

	for i := range order_list {
		order_list[i].Floor = -1
	}

	floor_queue_pointer := 0
	for floor, cr := range Cab_requests {
		if cr {
			elevio.SetButtonLamp(elevio.BT_Cab, floor, true)
			order_list[floor_queue_pointer] = data_structure.Order_list_t{floor, elevio.BT_Cab}
			floor_queue_pointer++
		}
	}

	for i, a := range delegated_hall_requests {
		for j, request := range a {
			if request {
				elevio.SetButtonLamp(elevio.ButtonType(j), i, true)
				order_list[floor_queue_pointer] = data_structure.Order_list_t{i, elevio.ButtonType(j)}
				floor_queue_pointer++
			}
		}
	}
	return order_list
}

func update_elevator_data(behaviour string, floor int, direction string, elevator_data data_structure.Elevator_data_t) data_structure.Elevator_data_t {
	elevator_data.Behaviour = behaviour
	elevator_data.Floor = floor
	elevator_data.Direction = direction
	return elevator_data
}
