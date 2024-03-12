package network

import (
	"Project/network/localip"
	"Project/network/peers"
	"Project/priority"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ConnectedPeers []int
var ReceivedElevators []int
var MasterElevators []int

func InitTimer(intId int) {

}

func InitNetwork(id string, intId int) {
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
		a := strings.Split(id, "-")
		intId, _ = strconv.Atoi(a[2])
	}

	ConnectedPeers = append(ConnectedPeers, intId)
	fmt.Printf("CONNECTEDPEERS: %v\n", ConnectedPeers)
}

func SlaveNetwork(id, port string, intId int, floor *int, broadcastedOrders chan []priority.Elevators, elevatorTransmit chan priority.Elevators, broadcastedMaster chan int, masterTimeout *time.Timer) {
	fmt.Println("Slave Started")

	var TransmitElevator = time.NewTimer(time.Second)
	var elevatorList []priority.Elevators
	var once = true
	for {
		select {
		case a := <-broadcastedOrders:
			elevatorList = a
			TransmitElevator.Reset(100 * time.Millisecond)
		case <-TransmitElevator.C:
			PushElevator(once, elevatorList, intId, floor, elevatorTransmit)
		case a := <-broadcastedMaster:
			masterTimeout.Reset(3*time.Second + time.Duration(intId*1000)*time.Millisecond)
			fmt.Printf("Master Elevator %d: Running\n", a)
			if !IdInList(a, MasterElevators) {
				MasterElevators = append(MasterElevators, a)
			}
			sort.Ints(MasterElevators)
			if len(MasterElevators) > 1 {
				RestartElevator(id, intId, port)
			}
		}
	}

}

func MasterNetwork(elevatorReceive <-chan priority.Elevators, peerUpdateCh <-chan peers.PeerUpdate, readyToBroadCast chan bool, receiveTimeout *time.Timer) {
	fmt.Printf("Current Elevator is Master\n")
	var elevator priority.Elevators

	for {
		select {
		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)
			UpdatePeers(p)
			priority.ElevatorList = ElevatorRedistribute(p, priority.ElevatorList)
		case a := <-elevatorReceive:
			elevator = a
			AssignElevator(elevator, readyToBroadCast, receiveTimeout)
		case <-receiveTimeout.C:
			AssignElevator(elevator, readyToBroadCast, receiveTimeout)
		}
	}
}

func PushElevator(once bool, elevatorList []priority.Elevators, intId int, floor *int, elevatorTransmit chan priority.Elevators) {
	if once {
		priority.UpdateCabOrders(elevatorList, intId)
		once = false
	}
	priority.UpdateOrders(elevatorList, intId)
	SendElevator(elevatorTransmit, intId, floor)
	priority.ClearSlaveCompleted(elevatorList, intId)
}

func SendElevator(elevatorTransmit chan<- priority.Elevators, intId int, floor *int) {
	a := priority.UpdateElevator(intId, floor)
	elevatorTransmit <- a
}
func BroadcastOrders(broadcastOrders chan []priority.Elevators, intId int, readyToBroadcast chan bool) {
	broadcastOrders <- priority.ElevatorList
	for {
		<-readyToBroadcast
		time.Sleep(100 * time.Millisecond) //200 good
		broadcastOrders <- priority.ElevatorList
	}
}

func UpdatePeers(p peers.PeerUpdate) {
	var IntPeer []int
	for _, v := range p.Peers {
		if strings.Contains(v, "peer") {
			a := strings.Split(v, "-")
			if a[2] != "[]" {
				b, _ := strconv.Atoi(a[2])
				IntPeer = append(IntPeer, b)
			}
		}
	}
	sort.Ints(IntPeer[:])
	ConnectedPeers = IntPeer
	fmt.Printf("%v", ConnectedPeers)
}

func ElevatorRedistribute(p peers.PeerUpdate, elevatorList []priority.Elevators) []priority.Elevators {
	var lostPeers []int
	for _, v := range p.Lost {
		if strings.Contains(v, "peer") {
			a := strings.Split(v, "-")
			if a[2] != "[]" {
				b, _ := strconv.Atoi(a[2])
				lostPeers = append(lostPeers, b)
			}
		}
	}

	elevatorList = priority.RedistributeDisconnectedOrders(lostPeers, elevatorList)
	return elevatorList
}

func ChooseMaster(broadcastedMaster chan int, toMaster chan bool, intId int, masterTimeout *time.Timer) {
	for {
		select {
		case <-masterTimeout.C:
			fmt.Printf("MasterTimeout\n")
			if ConnectedPeers[0] == intId {
				toMaster <- true
				return
			}
		}
	}
}

func BroadcastMaster(intId int, broadcastMaster chan int) {
	for {
		broadcastMaster <- intId
		time.Sleep(1*time.Second + time.Duration(intId*1000)*time.Millisecond)
	}
}

func AssignElevator(elevator priority.Elevators, readyToBroadCast chan bool, receiveTimeout *time.Timer) {
	receiveTimeout.Reset(1 * time.Second)
	UpdateElevatorList(elevator)
	ReceivedElevators = append(ReceivedElevators, elevator.Id)
	if ReceivedFromAll(ReceivedElevators, ConnectedPeers) {
		priority.ElevatorList = priority.DistributeOrders(priority.ElevatorList, ConnectedPeers)
		ReceivedElevators = nil
		readyToBroadCast <- true
	}
}

func IdInElevator(id int, elevatorList []priority.Elevators) bool {
	for _, b := range elevatorList {
		if b.Id == id {
			return true
		}
	}
	return false
}

func UpdateElevatorList(elevator priority.Elevators) {
	if len(priority.ElevatorList) <= 0 || !IdInElevator(elevator.Id, priority.ElevatorList) {
		priority.ElevatorList = append(priority.ElevatorList, elevator)
	}
	for i, v := range priority.ElevatorList {
		if v.Id == elevator.Id {
			priority.ElevatorList[i].ElevatorBehaviour = elevator.ElevatorBehaviour
			priority.ElevatorList[i].ElevatorDirection = elevator.ElevatorDirection
			priority.ElevatorList[i].Floor = elevator.Floor
			priority.ElevatorList[i].OrderRequest = elevator.OrderRequest
			priority.ElevatorList[i].Orders = elevator.Orders
			priority.ElevatorList[i].CompletedOrders = elevator.CompletedOrders
		}
	}
}

func ReceivedFromAll(receivedElevators, connectedPeers []int) bool {
	sum := 0
	for _, v := range receivedElevators {
		if IsConnected(v, connectedPeers) {
			sum++
		}
	}
	return sum >= len(connectedPeers)
}

func IsConnected(id int, connectedPeers []int) bool {
	for _, v := range connectedPeers {
		if id == v {
			return true
		}
	}
	return false
}

func IdInList(id int, list []int) bool {
	for _, b := range list {
		if b == id {
			return true
		}
	}
	return false
}

func deleteFromList(id int, list []int) []int {
	var index int
	for i, b := range list {
		if b == id {
			index = i
		}
	}
	return append(list[:index], list[index+1:]...)
}

func RestartElevator(id string, intId int, port string) {
	if MasterElevators[len(MasterElevators)-1] == intId {

		exec.Command("gnome-terminal", "--", "go", "run", "main.go", "-id="+id, "-port="+port).Run()
		panic("Multiple Masters!\n Restarting Elevator!\n")
	}
	fmt.Printf("Master Elevator %d: Terminated\n", MasterElevators[len(MasterElevators)-1])
	MasterElevators = deleteFromList(MasterElevators[len(MasterElevators)-1], MasterElevators)

}
