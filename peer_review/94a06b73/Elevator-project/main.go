package main

import (
	"Elevator-go/Elevator/elevator"
	"Elevator-go/Elevator/elevio"
	cf "Elevator-go/Elevator/type_"
	ctrl "Elevator-go/controller"
	intf "Elevator-go/network/networkInterface"
	"flag"
	"log"
)

func main() {

	/* run with 'go run main.go -id=elev-id -serverId=localhost:port' */

	var ServerId string
	/* give server id and elevator id from command line */
	flag.StringVar(&cf.LocalElevId, "id", "", "id of this peer")
	flag.StringVar(&ServerId, "serverId", "", "id of server")
	flag.Parse()

	if cf.LocalElevId[0:5] == ctrl.No_Id {
		log.Fatal("Use another id, do not use \"no-id\" string.")

	}

	/* connect to server */
	elevio.Init(ServerId, cf.NumFloors)

	/* Channels */
	ch_onRequestButtonPress := make(chan elevio.ButtonEvent) /* from driver to controller */
	ch_onFloorArrival := make(chan int)                      /* from driver to Elevator */
	ch_obstruction := make(chan bool)                        /* from driver to Elevator */
	ch_onStopButtonPress := make(chan bool)                  /* from driver to Elevator */

	ch_orderToExternalElevator := make(chan cf.OrderToExternalElev)
	ch_orderToLocalElevator := make(chan elevio.ButtonEvent)
	ch_orderFromExternalElevator := make(chan cf.OrderToExternalElev)
	ch_localElevatorStateToNtk := make(chan cf.LocalElevatorState)

	ch_ackToMaster := make(chan string)
	ch_ackFromElevs := make(chan string)

	/* elevio(driver) */
	go elevio.PollFloorSensor(ch_onFloorArrival)
	go elevio.PollObstructionSwitch(ch_obstruction)
	go elevio.PollButtons(ch_onRequestButtonPress)
	go elevio.PollStopButton(ch_onStopButtonPress)

	/* Controller */
	go ctrl.Elevator_controller(
		ch_onRequestButtonPress,
		ch_orderFromExternalElevator,
		ch_orderToLocalElevator,
		ch_orderToExternalElevator,
		ch_ackToMaster, ch_ackFromElevs)

	/* Interface to Network */
	go intf.Network_interface(
		ch_orderToExternalElevator,
		ch_orderFromExternalElevator,
		ch_localElevatorStateToNtk,
		ch_ackToMaster, ch_ackFromElevs)

	/* Elevator */
	go elevator.Elevator(
		ch_orderToLocalElevator,
		ch_onFloorArrival,
		ch_onStopButtonPress,
		ch_obstruction,
		ch_localElevatorStateToNtk)

	select {}
}
