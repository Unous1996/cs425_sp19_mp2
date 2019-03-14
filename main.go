package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"math/rand"
)

var (
	local_ip_address string
	localhost string
)

var (
	//server_address = "10.192.137.227"
	server_address = "172.22.156.52"
	server_portnumber = "8888" //Port number listen on
	gossip_fanout = 3
	serverhost string
)

var (
	working_chan chan bool
	introduce_chan chan string
	gossip_chan chan string
)

var (
	send_map map[string]*net.TCPConn
	send_map_mutex = sync.RWMutex{}
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

func gossip_transaction(){

	for {
		gossip_message := <-gossip_chan
		b := []byte(gossip_message)
		send_map_mutex.RLock()
		receivers := generateRandom(len(send_map) , gossip_fanout)
		send_map_mutex.RUnlock()
		count := 0

		send_map_mutex.RLock()
		for _, conn := range send_map {
			send_map_mutex.RUnlock()
			found := false
			for _, value := range receivers{
				if(count == value){
					found = true
				}
				send_map_mutex.RLock()
				break
			}
			if (found == false  || conn.RemoteAddr().String() == localhost) {
				send_map_mutex.RLock()
				continue
			}
			conn.Write(b)
			send_map_mutex.RLock()
		}
		send_map_mutex.RUnlock()
	}
}

func printTransaction(port_number string, xaction string){
	xaction_split := strings.Split(xaction, " ")
	fmt.Println(port_number + " = " + xaction_split[2])
}

func readMessage(port_number string, conn *net.TCPConn){

	buff := make([]byte, 256)
	for {
		j, err := conn.Read(buff)
		flag := checkErr(err)
		if flag == 0 {
			failed_remote := conn.RemoteAddr().String()
			if(len(strings.Split(failed_remote,":")[1]) != 5){
				fmt.Println(" The node with remote address " + failed_remote + "had failed")
			}
			send_map_mutex.Lock()
			delete(send_map, failed_remote)
			send_map_mutex.Unlock()
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
				working_chan <- true
			}

			if(line_split[0] == "TRANSACTION"){
				found := false
				for _, value := range holdback_transaction {
					if line == value{
						found = true
						break
					}
				}

				if(found == true){
					continue
				}

				printTransaction(port_number, line)
				holdback_transaction = append(holdback_transaction, line)
				gossip_chan <- line
			}

		}
	}
}

func addRemote(node_name string, ip_address string, port_number string){
	for {
		remotehost := <-introduce_chan

		send_map_mutex.RLock()
		if _, ok := send_map[remotehost]; ok {
			send_map_mutex.RUnlock()
			continue;
		}
		send_map_mutex.RUnlock()

		tcp_add, _ := net.ResolveTCPAddr("tcp", remotehost)
		remote_connection, err := net.DialTCP("tcp", nil, tcp_add)
		if err != nil {
			fmt.Println("Failed while dialing the remote node " + remotehost)
		}

		send_map_mutex.Lock()
		send_map[remotehost] = remote_connection
		send_map_mutex.Unlock()
		defer remote_connection.Close()
		go readMessage(port_number, remote_connection)

		send_map_mutex.RLock()
		for _, conn := range send_map{
			send_map_mutex.RUnlock()
			if(conn.RemoteAddr().String() != localhost){
				send_map_mutex.RLock()
				continue
			}
			conn.Write([]byte("INTRODUCE " + node_name + " " + ip_address + " " + port_number))
			send_map_mutex.RLock()
		}
		send_map_mutex.RUnlock()

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
		if conn == nil{
			continue
		}
		defer conn.Close()

		go readMessage(port_num, conn)
	}
}

func global_map_init(){
	send_map = make(map[string]*net.TCPConn)
}

func channel_init(){
	introduce_chan = make(chan string)
	gossip_chan = make(chan string)
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
	port_number := os.Args[2]
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

	localhost = local_ip_address + ":" + port_number
	connect_message := "CONNECT " + self_nodename + " " + local_ip_address + " " + port_number + "\n"
	fmt.Println("connect_message = ", connect_message)
	connect_message_byte := []byte(connect_message)

	go start_server(port_number)
	go addRemote(self_nodename, local_ip_address, port_number)

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
		go readMessage(port_number, server_connection)
		break
	}

	go gossip_transaction()

	<-working_chan
	fmt.Println("Process ended")
}