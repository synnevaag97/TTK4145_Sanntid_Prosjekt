package main

import (
	"Project/Driver-go/elevio"
	"Project/events"
	"Project/network"
	"Project/network/bcast"
	"Project/network/localip"
	"Project/network/peers"
	"Project/order"
	"Project/priority"
	"Project/statemachine"
	"flag"
	"fmt"
	"strconv"
	"time"
)

/* CONCURRENT ELEVATORS */

func main() {

	var Button elevio.ButtonEvent
	var Floor int
	var ObstructionSwitch bool
	var StopSwitch bool

	var id string
	var peerId string
	var portflag string
	localIP, _ := localip.LocalIP()

	flag.StringVar(&portflag, "port", "15657", "port of elevator")
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()
	var local = "localhost:" + portflag
	peerId = "peer-" + localIP + "-" + id
	intId, _ := strconv.Atoi(id)
	fmt.Printf("ID: %v\n", id)
	fmt.Printf("PORT: %v\n", portflag)
	ToMaster := make(chan bool)

	elevatorTransmit := make(chan priority.Elevators)
	elevatorReceive := make(chan priority.Elevators)
	broadcastOrders := make(chan []priority.Elevators)
	broadcastedOrders := make(chan []priority.Elevators)
	broadcastedMaster := make(chan int)
	broadcastMaster := make(chan int)
	readyToBroadCast := make(chan bool)

	peerTxEnable := make(chan bool)
	peerUpdateCh := make(chan peers.PeerUpdate)

	Buttons := make(chan elevio.ButtonEvent)
	Floors := make(chan int)
	Obstr := make(chan bool)
	Stop := make(chan bool)

	ToMovingState := make(chan bool, 2)
	ToDoorState := make(chan bool, 2)
	ToIdleState := make(chan bool, 2)

	ReceiveTimeout := time.NewTimer(time.Second)
	ReceiveTimeout.Stop()

	MasterTimeout := time.NewTimer(3*time.Second + time.Duration(intId*1000)*time.Millisecond)

	events.InitEvents()
	elevio.Init(local, order.NumFloors)
	network.InitNetwork(id, intId)
	elevio.SetDoorOpenLamp(false)

	statemachine.InitState(ToIdleState)
	priority.InitLastOrder()

	priority.UpdateElevator(intId, &Floor)

	go bcast.Transmitter(16569, elevatorTransmit)
	go bcast.Receiver(16570, broadcastedMaster)
	go bcast.Receiver(16571, broadcastedOrders)

	go peers.Transmitter(15640, peerId, peerTxEnable)
	go peers.Receiver(15640, peerUpdateCh)

	go elevio.PollButtons(Buttons)
	go elevio.PollFloorSensor(Floors)
	go elevio.PollObstructionSwitch(Obstr)
	go elevio.PollStopButton(Stop)
	go statemachine.StateHandler(ToMovingState, ToDoorState, ToIdleState, &Floor)
	go events.EventHandler(Buttons, Floors, Obstr, Stop, &Button, &Floor, &ObstructionSwitch, &StopSwitch)

	go network.ChooseMaster(broadcastedMaster, ToMaster, intId, MasterTimeout)
	go network.SlaveNetwork(id, portflag, intId, &Floor, broadcastedOrders, elevatorTransmit, broadcastedMaster, MasterTimeout)
	go priority.UpdateOrderLights()

	fmt.Printf("Current Elevator is Slave\n")

	<-ToMaster
	go bcast.Transmitter(16571, broadcastOrders)
	go bcast.Transmitter(16570, broadcastMaster)
	go bcast.Receiver(16569, elevatorReceive)

	go network.BroadcastMaster(intId, broadcastMaster)
	go network.MasterNetwork(elevatorReceive, peerUpdateCh, readyToBroadCast, ReceiveTimeout)
	go network.BroadcastOrders(broadcastOrders, intId, readyToBroadCast)

	select {}

}
