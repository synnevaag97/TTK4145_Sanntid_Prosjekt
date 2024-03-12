package supervisor

import (
	"Elevdriver/data_structure"
	"Elevdriver/network/findport"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"time"
)

func Supervisor(systemInfo data_structure.System_info_t, change_to_worker chan bool) {
	err_count := 0

	if systemInfo.Init {
		systemInfo.SuperPort, _ = findport.GetFreePort()
	}

	fmt.Println("Supervisor: using port: ", systemInfo.SuperPort)
	supervisorPort := fmt.Sprintf("localhost:%d", systemInfo.SuperPort)
	supervisorAddr, err := net.ResolveUDPAddr("udp", supervisorPort)
	if err != nil {
		fmt.Println("Supervisor: UDP resolve failed")
	}

	receive_conn, err := net.ListenUDP("udp", supervisorAddr)
	if err != nil {
		fmt.Println("Supervisor: connection refused")
	}

	// Start as a supervisior
	fmt.Println("Supervisor: Listen for barking...")
	for {
		p := make([]byte, 1024)
		receive_conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, _, err := receive_conn.ReadFromUDP(p)
		if err != nil {
			err_count++
			receive_conn.Close()
			fmt.Printf("Supervisor: UDP recevie failed %d times\n", err_count)
			break
		} else {
			// print(".")
		}
	}

	// We could not find a supervisor, start a new process
	commandArg := fmt.Sprintf("-id=\"%d\" -elevport=\"%d\" -superport=\"%d\"", systemInfo.Id, systemInfo.ElevPort, systemInfo.SuperPort)
	if runtime.GOOS == "windows" {
		fmt.Println("\nSupervisor: Windows OS detected")
		err = exec.Command("cmd", "/C", "start", "powershell", "go", "run", "main.go", commandArg).Run() // Windows
	} else {
		fmt.Println("\nSupervisor: Other OS detected")
		// err = exec.Command("gnome-terminal", "--", "go", "run", "main.go", commandArg).Run() //Linux
		err = exec.Command("gnome-terminal", "--", "go", "run", "main.go", "-init=false", "-elevport="+fmt.Sprint(systemInfo.ElevPort), "-id="+fmt.Sprint(systemInfo.Id), "-superport="+fmt.Sprint(systemInfo.SuperPort)).Run()
	}

	if err != nil {
		fmt.Println("\nSupervisor: Unable to reboot process, crashing...", err)
	}

	change_to_worker <- true
	fmt.Println("Supervisor: We start to bark...")
	send_conn, _ := net.DialUDP("udp", nil, supervisorAddr)
	for {
		_, _ = send_conn.Write([]byte("BARK!"))
		time.Sleep(time.Millisecond * 100)
		// print(".")
	}
}
