package watchdog

import (
	"Elevdriver/elevio"
	"fmt"
	"os"
	"time"
)

func Watchdog(time_out_time time.Duration, wdog_kick_chan chan bool) {

	wdog := time.NewTimer(time_out_time)
	for {
		select {
		case <-wdog_kick_chan:
			wdog.Reset(time_out_time)
		case <-wdog.C:
			fmt.Println("Watchdog timeeout, RESTARTING...")
			elevio.SetMotorDirection(elevio.MD_Stop)
			os.Exit(0)
		}
	}
}
