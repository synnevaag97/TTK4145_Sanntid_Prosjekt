package main

import (
	"Elevdriver/arbitration"
	"Elevdriver/backup"
	"Elevdriver/config"
	"Elevdriver/data_structure"
	"Elevdriver/distributor"
	"Elevdriver/elevator_network"
	"Elevdriver/elevio"
	"Elevdriver/supervisor"
	"flag"
	"fmt"
)

func main() {
	// Retreive system information
	var systemInfo data_structure.System_info_t
	flag.IntVar(&systemInfo.Id, "id", 0, "elevator id")
	flag.IntVar(&systemInfo.ElevPort, "elevport", 15657, "elevator port")
	flag.IntVar(&systemInfo.SuperPort, "superport", 80001, "Supervisor port")
	flag.IntVar(&systemInfo.PeerPort, "peerport", 38257, "peerport")
	flag.BoolVar(&systemInfo.Init, "init", false, "init")
	flag.Parse()
	fmt.Printf("Current settings: ID: %d, ElevPORT: %d, INIT: %t\n",
		systemInfo.Id, systemInfo.ElevPort, systemInfo.Init)

	// Channels
	change_to_worker := make(chan bool)
	arbitration_status_to_network_chan := make(chan data_structure.Arbitration_t)
	arbitration_status_to_distrbutor_chan := make(chan data_structure.Arbitration_t)
	cab_request_to_backup := make(chan data_structure.Elevator_data_t)
	stuck_send_chan := make(chan bool)
	order_send_chan := make(chan data_structure.Order_t, 2)
	order_receive_chan := make(chan data_structure.Order_t, 2)
	order_queue_send_chan := make(chan [config.ORDER_QUEUE_SIZE]data_structure.Order_t, 2)
	order_queue_receive_chan := make(chan [config.ORDER_QUEUE_SIZE]data_structure.Order_t, 2)
	hall_requests_send_chan := make(chan [config.NUM_ELEVATORS][config.NUM_FLOORS][2]bool, 2)
	hall_requests_receive_chan := make(chan [config.NUM_FLOORS][2]bool, 2)
	elevator_data_send_chan := make(chan data_structure.Elevator_data_t, 2)
	elevator_data_receive_chan := make(chan data_structure.Received_elevator_data_t, 2)

	// Supervisor
	go supervisor.Supervisor(systemInfo, change_to_worker)
	<-change_to_worker

	go backup.Backup(systemInfo,
		cab_request_to_backup)

	elevio.Init(systemInfo.ElevPort, config.NUM_FLOORS)

	go arbitration.Arbitration(systemInfo,
		arbitration_status_to_network_chan,
		arbitration_status_to_distrbutor_chan,
		stuck_send_chan)

	go elevator_network.Elevator_network(systemInfo,
		arbitration_status_to_network_chan,
		order_send_chan,
		order_receive_chan,
		order_queue_send_chan,
		order_queue_receive_chan,
		hall_requests_send_chan,
		hall_requests_receive_chan,
		elevator_data_send_chan,
		elevator_data_receive_chan)

	distributor.Distributor(systemInfo.Id,
		systemInfo,
		arbitration_status_to_distrbutor_chan,
		stuck_send_chan,
		elevator_data_receive_chan,
		elevator_data_send_chan,
		order_send_chan,
		order_receive_chan,
		order_queue_send_chan,
		order_queue_receive_chan,
		hall_requests_send_chan,
		hall_requests_receive_chan,
		cab_request_to_backup)
}
