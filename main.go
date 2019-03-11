package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
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
	introduce_chan chan string
)

var (
	read_map map[string]*net.TCPConn
	send_map map[string]*net.TCPConn
)

var (
	holdback_transaction []string
	pointer int
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

func multicast_transaction(){

	for {
		if(len(holdback_transaction) == 0){
			continue
		}

		b := []byte(holdback_transaction[pointer])

		for _, conn := range send_map {
			if conn.RemoteAddr().String() == localhost {
				continue
			}
			conn.Write(b)
		}

		pointer = (pointer + 1) % len(holdback_transaction)
		time.Sleep(2 * time.Second)
	}

}

func printTransaction(xaction string){
	xaction_split := strings.Split(xaction, " ")
	fmt.Println("XACT ID =", xaction_split[2])
}

func readMessage(conn *net.TCPConn){

	buff := make([]byte, 256)
	for {
		j, err := conn.Read(buff)
		flag := checkErr(err)
		if flag == 0 {
			fmt.Println(" #A node had failed")
			break
		}

		recevied_lines := strings.Split(string(buff[0:j]), "\n")
		for _, line := range recevied_lines {
			line_split := strings.Split(line, " ")
			if(line_split[0] == "Acknowledged"){
				fmt.Println("Received Acknowledged")
			}
			if(line_split[0] == "INTRODUCE"){
				fmt.Println("INTRODUCE Received")
				remotehost := line_split[2] + ":" + line_split[3]
				introduce_chan <- remotehost
			}

			if(line_split[0] == "TRANSACTION"){
				printTransaction(line)
				holdback_transaction = append(holdback_transaction, line)
			}
		}
	}
}

func addRemote(){
	for {
		remotehost := <-introduce_chan

		if _, ok := send_map[remotehost]; ok {
			continue;
		}

		tcp_add, _ := net.ResolveTCPAddr("tcp", remotehost)
		remote_connection, err := net.DialTCP("tcp", nil, tcp_add)
		if err != nil {
			fmt.Println("Failed while dialing the remote node " + remotehost)
		}
		send_map[remotehost] = remote_connection
		remote_connection.Write([]byte("Acknowledged"))
		go readMessage(remote_connection)
	}
	close(introduce_chan)
}

func start_server(port_num string) {

	tcp_addr, _ := net.ResolveTCPAddr("tcp", localhost)
	tcp_listen, err := net.ListenTCP("tcp", tcp_addr)

	if err != nil {
		fmt.Println("#Failed to listen on " + port_num)
	}

	fmt.Println("#Start listening on " + port_num)
	// Accept Tcp connection from other VMs
	for {
		conn, _ := tcp_listen.AcceptTCP()
		defer conn.Close()
		read_map[conn.RemoteAddr().String()] = conn


		go readMessage(conn)
	}
}

func global_map_init(){
	read_map = make(map[string]*net.TCPConn)
	send_map = make(map[string]*net.TCPConn)
}

func channel_init(){
	introduce_chan = make(chan string)
}

func main_init(){
	global_map_init()
	channel_init()
}

func main(){
	if len(os.Args) != 3 {
		fmt.Println(os.Stderr, "Incorrect number of parameters")
		os.Exit(1)
	}

	main_init()

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
	go addRemote()

	//Connect to server
	serverhost = server_address + ":" + server_portnumber
	for {
		tcp_add, _ := net.ResolveTCPAddr("tcp", serverhost)
		server_connection, err := net.DialTCP("tcp", nil, tcp_add)
		if err != nil {
			fmt.Println("#Failed to connect to the server")
			continue
		}

		defer server_connection.Close()
		server_connection.Write(connect_message_byte)
		read_map[serverhost] = server_connection
		go readMessage(server_connection)
		break
	}

	go multicast_transaction()

	<-working_chan
	fmt.Println("Shall not reach here")
}