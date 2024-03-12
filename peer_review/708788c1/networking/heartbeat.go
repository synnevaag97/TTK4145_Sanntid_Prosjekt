package networking

import (
	config "PROJECT-GROUP-[REDACTED]/config"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

var HeartBeatLogger bool = false

func heartBeatTransmitter(ch_req_ID chan int, ch_req_data chan Elevator_node,
	ch_hallCallsTot_updated chan<- [config.NUMBER_OF_FLOORS]HallCall,
	ch_deadlock_hb_trans chan<- bool) {

	var msg, date, clock, broadcast string
	var ID int = config.ELEVATOR_ID
	var node Elevator_node
	//Resolve transmit connection (broadcast)
	broadcast = "255.255.255.255:" + strconv.Itoa(config.HEARTBEAT_PORT)
	network, _ := net.ResolveUDPAddr("udp", broadcast)
	con, _ := net.DialUDP("udp", nil, network)

	fmt.Println("Networking: starting heartbeat transmision")
	ch_hallCallsTot_updated <- UpdateHallCallsTot(ch_req_ID, ch_req_data)
	timer := time.NewTimer(config.HEARTBEAT_TIME)
	for {
		<-timer.C
		timer.Reset(config.HEARTBEAT_TIME)
		//Sampling date and time, and making it nice european style
		year, month, day := time.Now().Date()
		date = strconv.Itoa(day) + "/" + month.String() + "/" + strconv.Itoa(year)
		hour, minute, second := time.Now().Clock()
		clock = strconv.Itoa(hour) + ":" + strconv.Itoa(minute) + ":" + strconv.Itoa(second)
		msg = date + " " + clock + "_"

		node = NodeGetData(ID, ch_req_ID, ch_req_data)

		//Generating the heartbeat message
		msg = msg + strconv.Itoa(ID) + "_"
		msg = msg + strconv.Itoa(node.Direction) + "_"
		msg = msg + strconv.Itoa(node.Destination) + "_"
		msg = msg + strconv.Itoa(node.Floor) + "_"
		msg = msg + strconv.Itoa(node.Status)

		for i := range node.HallCalls {
			var up, down int = 0, 0
			if node.HallCalls[i].Up {
				up = 1
			}
			if node.HallCalls[i].Down {
				down = 1
			}
			msg = msg + "_" + strconv.Itoa(up) + "_"
			msg = msg + strconv.Itoa(down)
		}

		if HeartBeatLogger {
			fmt.Println("Networking: sending HB message " + msg)
		}
		con.Write([]byte(msg))

		ch_hallCallsTot_updated <- UpdateHallCallsTot(ch_req_ID, ch_req_data)
		ch_deadlock_hb_trans <- true
	}
}

func heartBeathandler(
	ch_req_ID, ch_ext_dead chan int,
	ch_new_data, ch_take_calls chan<- int,
	ch_req_data, ch_write_data chan Elevator_node,
	ch_hallCallsTot_updated chan<- [config.NUMBER_OF_FLOORS]HallCall,
	ch_deadlock_hb_rec chan<- bool) {
	var node_data Elevator_node
	var ch_timerReset, ch_timerStop [config.NUMBER_OF_ELEVATORS]chan bool

	ch_heartbeatmsg := make(chan string)
	ch_found_dead := make(chan int)

	fmt.Println("Networking: HB starting listening thread")
	go heartbeatUDPListener(ch_heartbeatmsg)

	//Initiate heartbeat timers and channels for each elevator except for myself
	fmt.Println("Networking: HB starting timers")
	for i := 1; i <= config.NUMBER_OF_ELEVATORS; i++ {
		if i != config.ELEVATOR_ID {
			ch_timerReset[i-1] = make(chan bool)
			ch_timerStop[i-1] = make(chan bool)
			go heartbeatTimer(i, ch_found_dead, ch_timerReset[i-1], ch_timerStop[i-1])
		}
	}

	t := time.NewTimer(time.Second)
	t.Reset(time.Second)
	for {
		select {
		case <-t.C:
			t.Reset(time.Second)
			ch_deadlock_hb_rec <- true
		case msg := <-ch_heartbeatmsg:

			//Parsing/translating the received heartbeat message
			data := strings.Split(msg, "_")
			node_data.Last_seen = data[0]
			node_data.ID, _ = strconv.Atoi(data[1])
			node_data.Direction, _ = strconv.Atoi(data[2])
			node_data.Destination, _ = strconv.Atoi(data[3])
			node_data.Floor, _ = strconv.Atoi(data[4])
			node_data.Status, _ = strconv.Atoi(data[5])
			var k int = 0
			for i := range node_data.HallCalls {
				up, _ := strconv.Atoi(data[6+k])
				k++
				down, _ := strconv.Atoi(data[6+k])
				k++

				switch up {
				case 1:
					node_data.HallCalls[i].Up = true
				case 0:
					node_data.HallCalls[i].Up = false
				}

				switch down {
				case 1:
					node_data.HallCalls[i].Down = true
				case 0:
					node_data.HallCalls[i].Down = false
				}
			}
			if HeartBeatLogger {
				fmt.Println("Networking: Got heartbeat msg from elevator " + strconv.Itoa(node_data.ID) + ": " + msg)
				fmt.Println("Elevator " + strconv.Itoa(node_data.ID) + " at floor: " + strconv.Itoa(node_data.Floor))
			}

			ch_write_data <- node_data
			ch_timerReset[node_data.ID-1] <- true
			ch_hallCallsTot_updated <- UpdateHallCallsTot(ch_req_ID, ch_req_data)
			ch_new_data <- node_data.ID

		case msg_ID := <-ch_found_dead: //I found a dead elevator
			var msg, broadcast string

			ch_timerStop[msg_ID-1] <- true
			node_data = NodeGetData(msg_ID, ch_req_ID, ch_req_data)
			node_data.ID = msg_ID
			node_data.Status = 404
			ch_write_data <- node_data

			//Tell everyone that an elevator has died and that you are taking responsibility
			msg = "98_" + strconv.Itoa(msg_ID) + "_DEAD_" + strconv.Itoa(config.ELEVATOR_ID)
			broadcast = "255.255.255.255:" + strconv.Itoa(config.COMMAND_PORT)
			network, _ := net.ResolveUDPAddr("udp", broadcast)
			con, _ := net.DialUDP("udp", nil, network)
			con.Write([]byte(msg))
			con.Close()
			ch_new_data <- msg_ID
			ch_take_calls <- msg_ID
			fmt.Println("Networking: Elevator " + strconv.Itoa(msg_ID) + " is dead, redistributing his/her hall calls")
			ch_hallCallsTot_updated <- UpdateHallCallsTot(ch_req_ID, ch_req_data)
		case msg_ID := <-ch_ext_dead: //Set status to 404 and stop the timer
			node_data = NodeGetData(msg_ID, ch_req_ID, ch_req_data)
			node_data.Status = 404
			ch_write_data <- node_data
			ch_new_data <- msg_ID
			ch_timerStop[msg_ID-1] <- true
			ch_hallCallsTot_updated <- UpdateHallCallsTot(ch_req_ID, ch_req_data)

		}
	}
}

func heartbeatTimer(ID int, ch_foundDead chan<- int, ch_timerReset, ch_timerStop <-chan bool) {
	//Offset timeout based on elevator ID
	var time_TIMEOUT = config.HEARTBEAT_TIMEOUT + 100*time.Millisecond*time.Duration(config.ELEVATOR_ID)

	timer := time.NewTimer(time_TIMEOUT)
	timer.Stop()
	for {
		select {
		case <-timer.C:
			ch_foundDead <- ID
			timer.Stop()
		case <-ch_timerReset:
			timer.Reset(time_TIMEOUT)
		case <-ch_timerStop:
			timer.Stop()
		}
	}
}

func heartbeatUDPListener(ch_heartbeatmsg chan<- string) {
	buf := make([]byte, 1024)
	var msg string
	var port string = ":" + strconv.Itoa(config.HEARTBEAT_PORT)
	fmt.Println("Networking: Listening for HB-messages on port " + port)

	conn := DialBroadcastUDP(config.HEARTBEAT_PORT)

	for {
		n, _, _ := conn.ReadFrom(buf)
		msg = string(buf[0:n])
		data := strings.Split(msg, "_")
		ID, err := strconv.Atoi(data[1])

		//Checking weather the message is of the correct format and sending to Heartbeat Handler
		if err != nil {
			fmt.Println("Networking: got a bad heartbeat message " + msg)
			printError("Got error: ", err)
		} else if ID != config.ELEVATOR_ID && ID <= config.NUMBER_OF_ELEVATORS {
			ch_heartbeatmsg <- msg
		}
	}
}

//Returns an array of all the hallcalls currently being served
func UpdateHallCallsTot(ch_req_ID chan int, ch_req_data chan Elevator_node) (HallCallsTot [config.NUMBER_OF_FLOORS]HallCall) {
	for i := 1; i <= config.NUMBER_OF_ELEVATORS; i++ {
		Elevator := NodeGetData(i, ch_req_ID, ch_req_data)
		if Elevator.Status == 0 { //Ignore elevators with error
			for k := range Elevator.HallCalls {
				if Elevator.HallCalls[k].Up {
					HallCallsTot[k].Up = true
				}
				if Elevator.HallCalls[k].Down {
					HallCallsTot[k].Down = true
				}
			}
		}
	}
	return HallCallsTot
}
