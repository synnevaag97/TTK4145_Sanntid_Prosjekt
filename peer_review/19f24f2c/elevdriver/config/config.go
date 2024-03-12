package config

import "time"

const NUM_FLOORS int = 4

const NUM_ELEVATORS int = 3

const ORDER_QUEUE_SIZE int = NUM_FLOORS * 3 //Max individual orders: Hall_Up/down for each floor.

const MOTOR_TIMEOUT time.Duration = 10 * time.Second

const SEND_INTERVAL time.Duration = 500 * time.Millisecond

var Order_port_list [3]string = [3]string{"9143", "9243", "9343"}
var Order_ip string = "127.0.0.1"
