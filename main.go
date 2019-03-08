package main

import (
	"fmt"
	"net"
	"os"
)

var (
	local_ip_address string
)

var (
	server_address = "172.22.156.52"
	server_portnumber = "4444"
)

var (
	working_chan chan bool
)

func connect(message []byte, ch chan bool){

	for {
		tcp_add, _ := net.ResolveTCPAddr("tcp", "172.22.156.52:4444")
		conn, err := net.DialTCP("tcp", nil, tcp_add)
		if err != nil {
			fmt.Println("#Failed to connect to the server")
			continue
		}

		defer conn.Close()
		conn.Write(message)

		break
	}

	ch <- true
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println(os.Stderr, "Incorrect number of parameters")
		os.Exit(1)
	}

	self_nodename := os.Args[1]
	self_nodenumber := os.Args[2]

	//Get local ip address
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				local_ip_address = ipnet.IP.String()
				fmt.Println("#The local ip address is:", ipnet.IP.String())
			}
		}
	}

	connect_message := "CONNECT " + self_nodename + " " + local_ip_address + " " + self_nodenumber
	fmt.Println("connect_message = ", connect_message)
	connect_message_byte := []byte(connect_message)

	connect_chan := make(chan bool)
	go connect(connect_message_byte, connect_chan)
	<-connect_chan

	fmt.Println("From main goroutine: Finished sending connection")
	<-working_chan
	fmt.Println("Shall not reach here")
}
