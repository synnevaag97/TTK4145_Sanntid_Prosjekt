package main

import (
	"Controller-go/distributer"
	"flag"
	"fmt"
)

const NUM_FLOORS int = 4

//Starting point of the go program, Run the distributer and give a pulse into a file
func main() {
	fmt.Println("Starting up the elevator system!")

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")

	var port string
	flag.StringVar(&port, "port", "15657", "Define port to connect to")
	flag.Parse()

	reboot_chan := make(chan bool, 30)
	go distributer.RunDistributer(id, port, NUM_FLOORS, reboot_chan)
	for {
		select {
		case quit := <-reboot_chan:
			fmt.Println(quit)
			return
		}
	}
}
