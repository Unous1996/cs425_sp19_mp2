package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"math/rand"
	"encoding/csv"
)

var (
	local_ip_address string
)

var (
	//server_address = "10.192.137.227"
	server_address = "172.22.156.52"
	server_portnumber = "8888" //Port number listen on
	gossip_fanout = 3
	serverhost string
)

var (
	global_working_chan chan bool
	holdback_transaction map[string][]string
	program_start_time string
	writer *csv.Writer
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

func generateRandom(upper_bound int, num int) [] int{
	rand.Seed(time.Now().UnixNano())
	var result []int

	if(upper_bound <= num){
		for i := 0; i < upper_bound; i++ {
			result = append(result, i)
		}
		return result
	}
	
	for i := 0 ; i < num; i++ {
		for {
			appear := false
			temp := rand.Intn(upper_bound)
			for _, value := range result{
				if temp == value{
				appear = true
				}
			}

			if(appear == false){
				result = append(result, temp)
				break
			}
			temp += 1
		}
	}
	return result
}

func gossip_transaction(localhost string, send_map map[string]*net.TCPConn, gossip_chan chan string){
	for {
		gossip_message := <-gossip_chan
		b := []byte(gossip_message)
		receivers := generateRandom(len(send_map) , gossip_fanout)
		count := 0

		for _, conn := range send_map {
			found := false
			for _, value := range receivers{
				if(count == value){
					found = true
				}
				break
			}
			if (found == false  || conn.RemoteAddr().String() == localhost) {
				continue
			}
			conn.Write(b)
		}
	}
}

func printTransaction(port_num string, xaction string){
	xaction_split := strings.Split(xaction, " ")
	fmt.Println(port_num + " " + xaction_split[2])
	writer.Write([]string{port_num,xaction_split[2]})
	writer.Flush()
}

func readMessage(port_num string, conn *net.TCPConn, send_map map[string]*net.TCPConn, gossip_chan chan string, introduce_chan chan string){

	buff := make([]byte, 256)
	for {
		j, err := conn.Read(buff)
		flag := checkErr(err)
		if flag == 0 {
			failed_remote := conn.RemoteAddr().String()
			if(len(strings.Split(failed_remote,":")[1]) != 5){
				fmt.Println(" The node with remote address " + failed_remote + "had failed")
			}
			delete(send_map, failed_remote)
			break
		}

		recevied_lines := strings.Split(string(buff[0:j]), "\n")
		for _, line := range recevied_lines {
			line_split := strings.Split(line, " ")
		
			if(line_split[0] == "INTRODUCE"){
				remotehost := line_split[2] + ":" + line_split[3]
				introduce_chan <- remotehost
			}

			if(line_split[0] == "DIE" || line_split[0] == "QUIT"){
				os.Exit(1)
			}

			if(line_split[0] == "TRANSACTION"){
				found := false
				for _, value := range holdback_transaction[port_num] {
					if line == value {
						found = true
						break
					}
				}

				if(found == true){
					continue
				}

				printTransaction(port_num, line)
				holdback_transaction[port_num] = append(holdback_transaction[port_num], line)
				gossip_chan <- line
			}

		}
	}
}

func addRemote(node_name string, ip_address string, port_number string, send_map map[string]*net.TCPConn, gossip_chan chan string, introduce_chan chan string){
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
		defer remote_connection.Close()
		go readMessage(port_number, remote_connection, send_map, gossip_chan, introduce_chan)
		remote_connection.Write([]byte("INTRODUCE " + node_name + " " + ip_address + " " + port_number))
	}
	close(introduce_chan)
}

func start_server(port_num string, localhost string, send_map map[string]*net.TCPConn, gossip_chan chan string, introduce_chan chan string){
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
		go readMessage(port_num, conn, send_map, gossip_chan, introduce_chan)
	}
}

func node(nodename string, port_number string){
	send_map := make(map[string]*net.TCPConn)
	introduce_chan := make(chan string)
	gossip_chan := make(chan string)
	working_chan := make(chan string)

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

	localhost := local_ip_address + ":" + port_number
	connect_message := "CONNECT " + nodename + " " + local_ip_address + " " + port_number + "\n"
	fmt.Println("connect_message = ", connect_message)
	connect_message_byte := []byte(connect_message)

	go start_server(port_number, localhost, send_map, gossip_chan, introduce_chan)
	go addRemote(nodename, local_ip_address, port_number, send_map, gossip_chan, introduce_chan)

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
		go readMessage(port_number, server_connection, send_map, gossip_chan, introduce_chan)
		break
	}

	go gossip_transaction(localhost, send_map, gossip_chan)

	<-working_chan
	fmt.Println("Shall not reach here")
}

func main(){
	if len(os.Args) %2 != 1 {
		fmt.Println(os.Stderr, "Incorrect number of parameters")
		os.Exit(1)
	}

	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.MkdirAll("logs", os.ModePerm)
	} 

	program_start_time = time.Now().String()
	holdback_transaction = make(map[string][]string)
	//output_files = make(map[string]*File)

	file_name := program_start_time + ".csv"
	file, err := os.Create(file_name)
	
	if err != nil {
		panic(err)
	}

	writer = csv.NewWriter(file)
	writer.Write([]string{"Port Number","Transaction ID"})
	writer.Flush()

	defer file.Close()

	for i := 0; i < len(os.Args) / 2; i++ {
		node_name := os.Args[2*i+1]
		port_number := os.Args[2*i+2]
		holdback_transaction[port_number] = []string{}

		go node(node_name, port_number)
	}

	<-global_working_chan
	fmt.Println("Shall not reach here")
}