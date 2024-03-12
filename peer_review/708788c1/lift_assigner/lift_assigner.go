package ordering

import (
	"PROJECT-GROUP-[REDACTED]/config"
	elevio "PROJECT-GROUP-[REDACTED]/elevio"
	networking "PROJECT-GROUP-[REDACTED]/networking"
	"fmt"
	"math"
	"sort"
	"sync"
)

type score_tracker struct {
	score           int
	elevator_number int
}

type score_tracker_list []score_tracker

var elev_overview [config.NUMBER_OF_ELEVATORS]networking.Elevator_node
var number_of_alive_elevs int
var m sync.Mutex

//Checks and alerts the system whenever a heartbeat ping occurs
func heartbeatMonitor(
	ch_new_data chan int,
	ch_req_ID chan int,
	ch_req_data chan networking.Elevator_node,
) {
	for i := 1; i <= config.NUMBER_OF_ELEVATORS; i++ {
		if i != config.ELEVATOR_ID {
			elev_overview[i-1].ID = i
			elev_overview[i-1].Status = 2 //Status = 2: have not heard from it yet
		} else {
			elev_overview[i-1].ID = i
		}
	}
	for {
		id := <-ch_new_data
		elev_overview[id-1] = networking.NodeGetData(id, ch_req_ID, ch_req_data)                                 //updates elev_overview with the new data
		elev_overview[config.ELEVATOR_ID-1] = networking.NodeGetData(config.ELEVATOR_ID, ch_req_ID, ch_req_data) //update myself
	}
}

func PassToNetwork(
	ch_drv_buttons chan elevio.ButtonEvent,
	ch_new_order chan bool,
	ch_take_calls chan int,
	ch_command_elev chan elevio.ButtonEvent,
	ch_new_data chan int,
	ch_req_ID chan int,
	ch_req_data chan networking.Elevator_node,
) {
	go heartbeatMonitor(
		ch_new_data,
		ch_req_ID,
		ch_req_data,
	)
	fmt.Println("Lift assigner: starting up")
	for {
		select {
		case a := <-ch_drv_buttons: //Assigns a thread for each incomming call
			go callHandler(ch_command_elev, a, elev_overview)
		case death_id := <-ch_take_calls: //Assigns a thread for each redistributed call from dead elevator
			for floor := 0; floor < config.NUMBER_OF_FLOORS; floor++ {
				if elev_overview[death_id-1].HallCalls[floor].Up {
					go deathCallHandler(floor, 0, death_id, ch_drv_buttons, elev_overview)
				}
				if elev_overview[death_id-1].HallCalls[floor].Down {
					go deathCallHandler(floor, 1, death_id, ch_drv_buttons, elev_overview)
				}
			}
		}
	}
}

func callHandler(ch_command_elev chan elevio.ButtonEvent, a elevio.ButtonEvent, lighthouse [config.NUMBER_OF_ELEVATORS]networking.Elevator_node) {
	var placement [config.NUMBER_OF_ELEVATORS]score_tracker
	switch a.Button {
	case elevio.BT_HallUp:
		placement := masterTournament(a.Floor, int(elevio.MD_Up), placement, lighthouse)
		dir := int(elevio.MD_Up)
		SendToBestElevator(ch_command_elev, a, dir, lighthouse, placement)
	case elevio.BT_HallDown:
		placement := masterTournament(a.Floor, elevio.MD_Down, placement, lighthouse)
		dir := int(elevio.MD_Down)
		SendToBestElevator(ch_command_elev, a, dir, lighthouse, placement)
	case elevio.BT_Cab:
		ch_command_elev <- a
	}
}

func deathCallHandler(floor int, dir int, ID int, ch_drv_buttons chan elevio.ButtonEvent, lighthouse [config.NUMBER_OF_ELEVATORS]networking.Elevator_node) {
	var button_event elevio.ButtonEvent
	button_event.Floor = floor
	switch dir {
	case 0:
		button_event.Button = elevio.BT_HallUp
	case 1:
		button_event.Button = elevio.BT_HallDown
	}
	ch_drv_buttons <- button_event
}

//A function that scores all the elevators based on two inputs: floor and direction
func masterTournament(floor, direction int, placement [config.NUMBER_OF_ELEVATORS]score_tracker, lighthouse [config.NUMBER_OF_ELEVATORS]networking.Elevator_node) (return_placement [config.NUMBER_OF_ELEVATORS]score_tracker) {
	for i := 0; i < config.NUMBER_OF_ELEVATORS; i++ {
		placement[i].score = 1
		placement[i].elevator_number = i
	}
	for i := 0; i < config.NUMBER_OF_ELEVATORS; i++ {
		if !(lighthouse[i].Status != 0) {
			placement[i].score = calculateScore(placement, lighthouse[i])
			if (floor == lighthouse[i].Floor) && (lighthouse[i].Direction == 0) {
				placement[i].score = -1
			}
		} else { //Gives bad score if elevator is unreachable/unusable
			placement[i].score = 11
		}
	}
	return placement
}

func calculateScore(placement [config.NUMBER_OF_ELEVATORS]score_tracker, lighthouse networking.Elevator_node) (duration int) {
	var time int = 0
	switch lighthouse.Direction {
	case elevio.MD_Stop: //Elevator is idle so it is the best suited
		return time
	default:
		for i := 0; i < config.NUMBER_OF_FLOORS; i++ {
			if lighthouse.HallCalls[i].Up {
				time += int(math.Abs(float64(lighthouse.Floor - i)))
			}
			if lighthouse.HallCalls[i].Down {
				time += int(math.Abs(float64(lighthouse.Floor - i)))
			}
		}
		return time
	}
}

func SendToBestElevator(ch_command_elev chan elevio.ButtonEvent, a elevio.ButtonEvent, dir int, lighthouse [config.NUMBER_OF_ELEVATORS]networking.Elevator_node, placement [config.NUMBER_OF_ELEVATORS]score_tracker) {

	var temporary_placement [config.NUMBER_OF_ELEVATORS]score_tracker = sorting(placement)
	for i := 0; i < config.NUMBER_OF_ELEVATORS; i++ { //Cycle the scoreboard and attempt to send from best to worst
		if lighthouse[temporary_placement[i].elevator_number].ID == config.ELEVATOR_ID && lighthouse[temporary_placement[i].elevator_number].Status == 0 {
			fmt.Printf("Own elevator won\n")
			ch_command_elev <- a
			break
		} else if lighthouse[temporary_placement[i].elevator_number].Status == 0 {
			fmt.Printf("Trying to send to elevator %d\n", placement[i].elevator_number)
			m.Lock()
			success := networking.SendCommand(lighthouse[temporary_placement[i].elevator_number].ID, a.Floor, dir)
			m.Unlock()
			if success {
				fmt.Printf("Managed to send to elevator %d\n", placement[i].elevator_number)
				break
			}
		}
		if i == config.NUMBER_OF_ELEVATORS-1 { //Send to self if no one is avaliable
			ch_command_elev <- a
		}
	}
}

//Quicksort for struct
func sorting(placement [config.NUMBER_OF_ELEVATORS]score_tracker) (return_placement [config.NUMBER_OF_ELEVATORS]score_tracker) {
	sort.Sort(score_tracker_list(placement[:]))
	return placement
}

//Functions for quicksort
//==============================================================
func (temp_score score_tracker_list) Len() int {
	return len(temp_score)
}

func (temp_score score_tracker_list) Less(i, j int) bool {
	return temp_score[i].score < temp_score[j].score
}

func (temp_score score_tracker_list) Swap(i, j int) {
	temp_score[i], temp_score[j] = temp_score[j], temp_score[i]
}

//==============================================================
