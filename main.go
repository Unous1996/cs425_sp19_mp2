package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

var (
	local_ip_address string
	localhost string
)

var (
	//server_address = "10.192.137.227"
	server_address = "172.22.156.52"
	server_portnumber = "8888" //Port number listen on
	serverhost string
)

var (
	working_chan chan bool
)

func checkErr(err error) int {
	if err != nil {
		if err.Error() == "EOF" {
			//fmt.Println(err)
			return 0
		}
		fmt.Println(err)
		return -1
	}
	return 1
}

func readMessage(conn *net.TCPConn){
	fmt.Println("Begin read message")
	buff := make([]byte, 256)
	for {
		j, err := conn.Read(buff)
		flag := checkErr(err)
		if flag == 0 {
			fmt.Println(" #A node had failed")
			break
		}

		fmt.Println("Recevied string = ", string(buff[0:j]))
		recevied_string_spilt := strings.Split(string(buff[0:j]), " ")

		if recevied_string_spilt[0] == "INTRODUCE" {
			fmt.Println("Recevied an INTRODUCE")
		}

		if recevied_string_spilt[0] == "TRANSACTION" {
			fmt.Println("Recevied an TRANSACTION")
		}
	}
}

func start_server(port_num string){
	tcp_addr, _ := net.ResolveTCPAddr("tcp", localhost)
	tcp_listen, err := net.ListenTCP("tcp", tcp_addr)

	if err != nil {
		fmt.Println("#Failed to listen on " + port_num)
	}

	fmt.Println("#Start listening on " + port_num)
	for {
		conn, err := tcp_listen.AcceptTCP()
		if err != nil{
			fmt.Println("Error in AcceptTCP")
		}
		fmt.Println("Accepted a TCP")
		defer conn.Close()
		go readMessage(conn)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println(os.Stderr, "Incorrect number of parameters")
		os.Exit(1)
	}

	self_nodename := os.Args[1]
	self_server_port_number := os.Args[2]

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

	localhost = local_ip_address + ":" + self_server_port_number
	connect_message := "CONNECT " + self_nodename + " " + local_ip_address + " " + self_server_port_number + "\n"
	fmt.Println("connect_message = ", connect_message)
	connect_message_byte := []byte(connect_message)

	go start_server(self_server_port_number)

	serverhost = server_address + ":" + server_portnumber
	for {
		tcp_add, _ := net.ResolveTCPAddr("tcp", serverhost)
		conn, err := net.DialTCP("tcp", nil, tcp_add)
		if err != nil {
			fmt.Println("#Failed to connect to the server")
			continue
		}

		defer conn.Close()
		conn.Write(connect_message_byte)

		break
	}

	<-working_chan
	fmt.Println("Shall not reach here")
}
