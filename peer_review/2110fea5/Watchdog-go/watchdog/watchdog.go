package watchdog

import (
	"Driver-go/elevio"
	"Types-go/Type-Msg/messages"
	"fmt"
	"time"
)

//Watchdog function, verify that orders are executed within a specific time and inform/bark on a channel if not.
//If the order was from our own node, we reboot the program.
func RunWatchdog(
	numFloors int,
	Id string,
	assigned_hall_requests <-chan map[string][][2]bool,
	hall_request_completed <-chan messages.ReqCompleteMsg,
	hall_request_expired chan<- messages.ReqCompleteMsg,
	cab_request_to_watchdog <-chan elevio.ButtonEvent,
	cab_request_completed <-chan messages.ReqCompleteMsg,
	database_timeout_expired chan<- bool,
	database_arrived <-chan bool,
	//initiate_single_elevator chan<- bool,
	start_timer <-chan bool,
	reboot_chan chan<- bool,
) {

	order_list := make(map[elevio.ButtonEvent]messages.ReqCompleteMsg)
	hall_requests := make(map[string][][2]bool)
	cab_requests := make([]bool, numFloors)
	initiation_database_arrived := false

	request_timer_expired := make(chan messages.ReqCompleteMsg, 20)
	init_timer_expired := make(chan bool, 10)

	for {
		select {

		case <-start_timer:
			go initiation_timer(init_timer_expired)

		case <-database_arrived:
			initiation_database_arrived = true

		case <-init_timer_expired:
			if !initiation_database_arrived {
				database_timeout_expired <- true
			}

		//If we receive a new hall request
		case updated_hall_requests := <-assigned_hall_requests:
			number_of_node := len(updated_hall_requests)

			if number_of_node >= 1 {
				for k := range updated_hall_requests {
					if hall_requests[k] == nil {
						hall_requests[k] = make([][2]bool, numFloors)
					}
					for f := 0; f < numFloors; f++ {
						for b := 0; b < 2; b++ {
							if hall_requests[k][f][b] != updated_hall_requests[k][f][b] && updated_hall_requests[k][f][b] {
								order := messages.Create_ReqCompleteMsg(k, elevio.ButtonEvent{Floor: f, Button: elevio.ButtonType(b)})
								go request_timer(number_of_node, order, request_timer_expired)
								order_list[order.Request] = order
							}
						}
					}
				}
				hall_requests = updated_hall_requests
			}

		//If the timer expired
		case order_timeout := <-request_timer_expired:
			check_order_completed(Id, order_timeout, &order_list, hall_requests, hall_request_expired, reboot_chan)

		//If a request has been completed
		case request_completed := <-hall_request_completed:
			if thisR, ok := hall_requests[request_completed.Id]; ok {
				thisR[request_completed.Request.Floor][request_completed.Request.Button] = false
				hall_requests[request_completed.Id] = thisR

				//remove the order from the map
				delete(order_list, request_completed.Request)
			}

		//If we receive a new cab request
		case updated_cab := <-cab_request_to_watchdog:
			if !(cab_requests[updated_cab.Floor]) {
				cab_requests[updated_cab.Floor] = true

				order := messages.Create_ReqCompleteMsg("", updated_cab)
				go request_timer(1, order, request_timer_expired)
				order_list[order.Request] = order

			}

		//If a cab request has been completed
		case removed_cab := <-cab_request_completed:
			cab_requests[removed_cab.Request.Floor] = false
			delete(order_list, removed_cab.Request)
		}
	}
}

//Timer function
func request_timer(number_of_node int, order messages.ReqCompleteMsg, timeout chan<- messages.ReqCompleteMsg) {
	time.Sleep(time.Duration(50-(10*number_of_node)) * time.Second)
	timeout <- order
}

func initiation_timer(timeout chan<- bool) {
	time.Sleep(1 * time.Second)
	timeout <- true
}

//Compare the updated hall request with an order to check if it has been taken
func check_order_completed(
	Id string,
	order messages.ReqCompleteMsg,
	order_list *map[elevio.ButtonEvent]messages.ReqCompleteMsg,
	updated_hall_request map[string][][2]bool,
	hall_request_expired chan<- messages.ReqCompleteMsg,
	reboot_chan chan<- bool) {

	if order.Request.Button == elevio.BT_Cab {
		_, ok := (*order_list)[order.Request]
		if ok && (order.Timestamp == (*order_list)[order.Request].Timestamp) {
			fmt.Println("=============================")
			fmt.Print("  Cab order expired: ")
			fmt.Println(order)
			fmt.Println("=============================")
			fmt.Println("Killing program from watchdog (CAB)")
			reboot_chan <- true
		}

	} else {

		_, ok := updated_hall_request[order.Id]
		if ok {
			if updated_hall_request[order.Id][order.Request.Floor][order.Request.Button] {
				_, ok := (*order_list)[order.Request]
				if ok && (order.Timestamp == (*order_list)[order.Request].Timestamp) {
					fmt.Println("=============================")
					fmt.Print("  Hall order expired: ")
					fmt.Println(order)
					fmt.Println("=============================")
					delete(updated_hall_request, order.Id)
					for k := range *order_list {
						if (*order_list)[k].Id == order.Id {
							delete((*order_list), k)
						}
					}
					if order.Id == Id {
						fmt.Println("Killing program from watchdog (HALL)")
						reboot_chan <- true

					} else {
						hall_request_expired <- order
					}
				}
			}
		}
	}
}
