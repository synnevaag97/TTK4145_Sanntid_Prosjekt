package elevator_network

import (
	"Elevdriver/config"
	"Elevdriver/data_structure"
	"Elevdriver/watchdog"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xtaci/kcp-go/v5"
)

const (
	_             = iota
	order         = iota
	order_queue   = iota
	hall_request  = iota
	elevator_data = iota
)

type Network_message_t struct {
	UUID        string
	SenderId    int
	ReceiverId  int
	PayloadType string
	Payload     string
}

const watchdog_timeout = 10 * time.Second

var Elevator_network_debug = false

func Elevator_network(systemInfo data_structure.System_info_t,
	arbitration_status_chan chan data_structure.Arbitration_t,
	order_send_chan chan data_structure.Order_t,
	order_receive_chan chan data_structure.Order_t,
	order_queue_send_chan chan [config.ORDER_QUEUE_SIZE]data_structure.Order_t,
	order_queue_receive_chan chan [config.ORDER_QUEUE_SIZE]data_structure.Order_t,
	hall_requests_send_chan chan [config.NUM_ELEVATORS][config.NUM_FLOORS][2]bool,
	hall_requests_recieve_chan chan [config.NUM_FLOORS][2]bool,
	elevator_data_send_chan chan data_structure.Elevator_data_t,
	elevator_data_receive_chan chan data_structure.Received_elevator_data_t) {

	watchdog_feed := make(chan bool)
	network_message_rx_chan := make(chan Network_message_t)

	var network_message_tx_chans = [config.NUM_ELEVATORS]chan Network_message_t{}
	var server_disconnected_chans = [config.NUM_ELEVATORS]chan bool{}
	var arbitration_status data_structure.Arbitration_t

	go watchdog.Watchdog(watchdog_timeout, watchdog_feed)
	go server(systemInfo, network_message_rx_chan)

	//Start one client for each server other than our own
	for elev := 0; elev < config.NUM_ELEVATORS; elev++ {
		if elev != systemInfo.Id {
			network_message_tx_chans[elev] = make(chan Network_message_t)
			server_disconnected_chans[elev] = make(chan bool)
			go client(systemInfo, config.Order_port_list[elev],
				network_message_tx_chans[elev],
				server_disconnected_chans[elev])
		}
	}

	for {
		select {
		case <-time.After(100 * time.Millisecond):
			watchdog_feed <- true
		case arbitration := <-arbitration_status_chan:
			//Tell the client that a server has disconnected
			for i := 0; i < config.NUM_ELEVATORS; i++ {
				if arbitration.Alive_list[i] == true && arbitration_status.Alive_list[i] == false && i != systemInfo.Id {
					server_disconnected_chans[i] <- true
				}
			}
			arbitration_status = arbitration

		case network_message_rx := <-network_message_rx_chan:
			switch network_message_rx.PayloadType {
			case "order":
				var payload data_structure.Order_t
				err := json.Unmarshal([]byte(network_message_rx.Payload), &payload)
				if err != nil {
					fmt.Println("Elevator_network: Failed to decode order")
				} else {
					order_receive_chan <- payload
				}
			case "order_queue":
				var payload [config.ORDER_QUEUE_SIZE]data_structure.Order_t
				err := json.Unmarshal([]byte(network_message_rx.Payload), &payload)
				if err != nil {
					fmt.Println("Elevator_network: Failed to decode order_queue")
				} else {
					order_queue_receive_chan <- payload
				}

			case "hall_requests":
				var payload [config.NUM_FLOORS][2]bool
				err := json.Unmarshal([]byte(network_message_rx.Payload), &payload)
				if err != nil {
					fmt.Println("Elevator_network: Failed to decode hall_requests")
				} else {
					hall_requests_recieve_chan <- payload
				}
			case "elevator_data":
				var payload data_structure.Elevator_data_t
				err := json.Unmarshal([]byte(network_message_rx.Payload), &payload)
				if err != nil {
					fmt.Println("Elevator_network: Failed to decode elevator_data")
				} else {
					offload := data_structure.Received_elevator_data_t{network_message_rx.SenderId, payload}
					elevator_data_receive_chan <- offload
				}
			}
		case order_send := <-order_send_chan:
			//from slave to master
			for id, alive := range arbitration_status.Alive_list {
				if alive && id != systemInfo.Id {
					order_json, _ := json.Marshal(order_send)
					network_message_tx_chans[id] <- network_message_tx_encoder(systemInfo.Id,
						id, "order", string(order_json))
					break //master found
				}
			}
		case order_queue_send := <-order_queue_send_chan:
			//master send to slaves
			for id, alive := range arbitration_status.Alive_list {
				if alive && id != systemInfo.Id {
					order_queue_json, _ := json.Marshal(order_queue_send)
					network_message_tx_chans[id] <- network_message_tx_encoder(systemInfo.Id,
						id, "order_queue", string(order_queue_json))
				}
			}

		case hall_requests_send := <-hall_requests_send_chan:
			//master send to slaves
			for id, alive := range arbitration_status.Alive_list {
				if alive && id != systemInfo.Id {
					hall_requests_json, _ := json.Marshal(hall_requests_send[id])
					network_message_tx_chans[id] <- network_message_tx_encoder(systemInfo.Id,
						id, "hall_requests", string(hall_requests_json))
				}
			}

		case elevator_data_send := <-elevator_data_send_chan:
			//slave send to master
			for id, alive := range arbitration_status.Alive_list {
				if alive && id != systemInfo.Id {
					elevator_data_json, _ := json.Marshal(elevator_data_send)
					network_message_tx_chans[id] <- network_message_tx_encoder(systemInfo.Id,
						id, "elevator_data", string(elevator_data_json))
					break //master found
				}
			}
		}
	}
}

/* Function: genUUID
   32 character long id used to identify messages
   format : 123e4567-e89b-12d3-a456-426614174000
*/
func genUUID() string {
	id := uuid.New().String()
	return id
}

func client(systemInfo data_structure.System_info_t, port string,
	network_message_tx_chan chan Network_message_t,
	server_disconnected_chan chan bool) {

	for {
		destinationAddr := fmt.Sprintf("%s:%s", config.Order_ip, port)
		conn, err := kcp.Dial(destinationAddr)
		if err != nil {
			fmt.Println("Elevator_network: Failed to connect:", err.Error())
			fmt.Println("Elevator_network: Trying to reconnect...")
			time.Sleep(time.Millisecond * time.Duration(1000))
		} else {
			connected := true
			fmt.Printf("Connected to port: %s\n", port)
			for connected {
				select {
				case network_message_tx := <-network_message_tx_chan:
					defer conn.Close()
					msg_tx := transmitter_encoder(network_message_tx)
					_, err := conn.Write([]byte(msg_tx))
					if err != nil {
						fmt.Println("Elevator_network: Write to server failed:", err.Error())
						fmt.Println("Elevator_network: Trying to reconnect...")
						connected = false
					}
				case <-server_disconnected_chan:
					connected = false
				}
			}
		}
	}
}

/* Function: read_from_client
   This function takes a connection and read the data and send
   it back to the network_message_rx_chan channel
*/
func read_from_client(conn *kcp.UDPSession,
	systemInfo data_structure.System_info_t,
	network_message_rx_chan chan Network_message_t) {

	request := make([]byte, 1024)
	defer conn.Close()

	for {
		msg_rx_len, err := conn.Read(request)
		if err != nil {
			print_err(err)
			break
		}
		if msg_rx_len == 0 {
			break
		}
		msg_rx_raw := strings.TrimSpace(string(request[:msg_rx_len]))
		msg_rx := reader_decoder(msg_rx_raw, msg_rx_len)
		network_message_rx_chan <- msg_rx
	}
}

/* Function: Server
   Each elevator will create its own server that will listen to incomming messages
   from other elevators. It will generate a new read_from_client go runtine for each
   incomming client.
*/
func server(systemInfo data_structure.System_info_t,
	network_message_rx_chan chan Network_message_t) {

	addr := fmt.Sprintf("%s:%s", config.Order_ip, config.Order_port_list[systemInfo.Id])
	listener, _ := kcp.ListenWithOptions(addr, nil, 0, 0)

	for {
		conn, err := listener.AcceptKCP()
		if err != nil {
			continue
		}
		go read_from_client(conn, systemInfo, network_message_rx_chan)
	}
}

/* Function: transmitter_encoder [private]
   Takes a Network_mssage_t object and convert it to a string that is sent over TCP.
   Each part of the message is split using a seprator <|> and ended with <\n>
*/
func transmitter_encoder(msg_tx_in Network_message_t) (msg_tx_out string) {
	print_send_message(msg_tx_in)
	msg_tx_out = fmt.Sprintf("%s|%d|%d|%s|%s|\n",
		msg_tx_in.UUID,
		msg_tx_in.SenderId,
		msg_tx_in.ReceiverId,
		msg_tx_in.PayloadType,
		msg_tx_in.Payload)

	return msg_tx_out
}

/* Function: reader_decoder [private]
   Takes a message received over TCP and coverts it to a object of type Network_message_t
   Each part of the message is located and extracted by looking for the <|> seperator
*/
func reader_decoder(msg_rx_in string, msg_rx_len int) (msg_rx_out Network_message_t) {
	msg_rx_out.UUID = strings.Split(msg_rx_in, "|")[0]
	msg_rx_out.SenderId, _ = strconv.Atoi(strings.Split(msg_rx_in, "|")[1])
	msg_rx_out.ReceiverId, _ = strconv.Atoi(strings.Split(msg_rx_in, "|")[2])
	msg_rx_out.PayloadType = strings.Split(msg_rx_in, "|")[3]
	msg_rx_out.Payload = strings.Split(msg_rx_in, "|")[4]
	print_receive_message(msg_rx_out)

	return msg_rx_out
}

/* Function: network_message_tx_encoder [private]
   Takes the message parameters and gives them to a object of type Network_message_t
   Generates a unique UUID for each message that is mainly used to track messages for debugging
*/
func network_message_tx_encoder(senderId int, receiverId int, PayloadType string, Payload string) (msg_tx_out Network_message_t) {
	msg_tx_out.UUID = genUUID()
	msg_tx_out.SenderId = senderId
	msg_tx_out.ReceiverId = receiverId
	msg_tx_out.PayloadType = PayloadType
	msg_tx_out.Payload = Payload
	return msg_tx_out
}

/* Enable/disable infomation printing for debugging */
func Network_debugging(network_debugging bool) {
	Elevator_network_debug = network_debugging
}

/*======================================================================
  =                                                                    =
  =        Some nice formating of printing messages............        =
  =                                                                    =
  ======================================================================*/

func print_send_message(network_message Network_message_t) {
	if Elevator_network_debug == true {
		fmt.Printf("Elevator_network: sending %s with msg_id %s from %d to %d\n",
			network_message.PayloadType,
			network_message.UUID,
			network_message.SenderId,
			network_message.ReceiverId)
	}
}

func print_receive_message(network_message Network_message_t) {
	if Elevator_network_debug == true {
		fmt.Printf("Elevator_network: received %s with msg_id %s from %d to %d\n",
			network_message.PayloadType,
			network_message.UUID,
			network_message.SenderId,
			network_message.ReceiverId)
	}
}

func print_tcp_read(message_rx Network_message_t, len int) {
	if Elevator_network_debug == true {
		fmt.Printf("Elevator_network: received data from %d, to %d, with message length %d\n",
			message_rx.SenderId,
			message_rx.ReceiverId,
			len)
	}
}

func print_err(err error) {
	if Elevator_network_debug == true {
		fmt.Println("Elevator_network: ", err)
	}
}
