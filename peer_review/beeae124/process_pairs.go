package main

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"time"
	"os"
)

func processPairs(
	elevatorPort 								string, 
	elevatorId 									string,
	channel_processPairs_batonPass_1_2 	 chan<- string,
	channel_processPairs_batonPass_4_1 	 <-chan string) {

	//Assign udp-port
	udpPort, _ := strconv.Atoi(elevatorPort)
	udpPort = udpPort + 10
	
	ServerConn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: []byte{0, 0, 0, 0}, Port: udpPort, Zone: ""})

	buffer := make([]byte, 32)
	var processPairsBaton = "baton"
	var processPairsAlivePing string
	processPairsWDT := time.NewTimer(4 * time.Second)
	processPairsWDT.Stop()

	//Secondary ////////////////////////
	for {
		ServerConn.SetReadDeadline(time.Now().Add(6 * time.Second))
		size, _, err := ServerConn.ReadFromUDP(buffer[0:])
		if err == nil {
			//Receive the last elevator from primary
			processPairsAlivePing = string(buffer[:size])
			fmt.Println("Received alive-ping from primary:", processPairsAlivePing)
		} else {
			fmt.Println("timed out")
			ServerConn.Close()
			break
		}
	}

	//Primary /////////////////////////
	exec.Command("gnome-terminal", "--", "./Elevator" , elevatorPort, elevatorId).Run() // Spawn backup

	//Transmit on UDP 'localhost' 127.0.0.1
	Conn, _ := net.DialUDP("udp", nil, &net.UDPAddr{IP: []byte{127, 0, 0, 1}, Port: udpPort, Zone: ""})

	processPairsWDT.Reset(2 * time.Second)
	fmt.Println("Primary: First pass of baton from processPairs")
	channel_processPairs_batonPass_1_2 <- processPairsBaton
	for {
		select{
		//Receive baton from daisy-chain
		case <-channel_processPairs_batonPass_4_1:
			//Reset WDT
			processPairsWDT.Reset(3 * time.Second)

			//Primary needs to send the elevator to secondary over UDP as long as it is alive!
			Conn.Write([]byte("Primary-alive"))

			//Passing a baton in a daisy-chain of channels through 'fsm', 
			//'updateSharedDataWithHallRequests' and 'network_peerUpdate' go routines 
			//before resetting the WDT.
			channel_processPairs_batonPass_1_2 <- processPairsBaton

		case <-processPairsWDT.C:
			//Kill the primary
			processID_int := os.Getpid()
			processID := strconv.Itoa(processID_int)
			fmt.Println("Kill pID: ", processID)
			exec.Command("kill", "-9", processID).Run()
		default:
			break
		}
		time.Sleep(1 * time.Second)
	}
}
