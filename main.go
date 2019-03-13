package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"math/rand"
	"encoding/csv"
	"strconv"
	"sync"
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
	holdback_transaction_mutex = sync.RWMutex{}
	program_start_time string
	write_to_file bool
)

var(
	cleanup_chan chan os.Signal
	thanos_chan chan bool
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

func gossip_transaction(localhost string, send_map map[string]*net.TCPConn, send_map_mutex sync.RWMutex,  gossip_chan chan string){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
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

func printTransaction(port_num string, xaction string) string {
	xaction_split := strings.Split(xaction, " ")
	
	time_string := xaction_split[1]
	time_float,_ := strconv.ParseFloat(time_string,64)
	currt := time.Now()
	current_float := float64(currt.UTC().UnixNano()) / 1000.0 / 1000.0 / 1000.0
	time_difference := current_float - time_float
	time_difference_string := fmt.Sprintf("%f", time_difference)
	return_string := port_num + " " + xaction_split[2] + " " + time_difference_string
	fmt.Println(return_string)
	return return_string
}

func readMessage(port_num string, conn *net.TCPConn, send_map map[string]*net.TCPConn, send_map_mutex sync.RWMutex, gossip_chan chan string, introduce_chan chan string){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	buff := make([]byte, 256)
	for {
		j, err := conn.Read(buff)
		flag := checkErr(err)
		if flag == 0 {
			failed_remote := conn.RemoteAddr().String()
			/*
			if(len(strings.Split(failed_remote,":")[1]) != 5) {
				fmt.Println(" The node with remote address " + failed_remote + "had failed")
			}
			*/
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
				thanos_chan <- true
			}

			if(line_split[0] == "TRANSACTION" && len(line_split) == 6){
				found := false
				holdback_transaction_mutex.RLock()
				for _, stored_transaction := range holdback_transaction[port_num] {
					holdback_transaction_mutex.RUnlock()
					stored_transaction_split := strings.Split(stored_transaction, " ")
					if line_split[2] == stored_transaction_split[1] {
						found = true
						holdback_transaction_mutex.RLock()
						break
					}
					holdback_transaction_mutex.RLock()
				}
				holdback_transaction_mutex.RUnlock()

				if(found == true) {
					continue
				}

				holdback_message := printTransaction(port_num, line)
				holdback_transaction_mutex.Lock()
				fmt.Println("holdback_message = ", holdback_message)
				holdback_transaction[port_num] = append(holdback_transaction[port_num], holdback_message)
				holdback_transaction_mutex.Unlock()
				gossip_chan <- line
			}
		}
	}
}

func addRemote(node_name string, ip_address string, port_number string, send_map map[string]*net.TCPConn, send_map_mutex sync.RWMutex, gossip_chan chan string, introduce_chan chan string){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
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
			continue
		}

		send_map_mutex.Lock()
		send_map[remotehost] = remote_connection
		send_map_mutex.Unlock()

		defer remote_connection.Close()
		go readMessage(port_number, remote_connection, send_map, send_map_mutex, gossip_chan, introduce_chan)
		remote_connection.Write([]byte("INTRODUCE " + node_name + " " + ip_address + " " + port_number))
	}
	close(introduce_chan)
}

func start_server(port_num string, localhost string, send_map map[string]*net.TCPConn, send_map_mutex sync.RWMutex,gossip_chan chan string, introduce_chan chan string){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
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

		go readMessage(port_num, conn, send_map, send_map_mutex, gossip_chan, introduce_chan)
	}
}

func node(nodename string, port_number string){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	send_map := make(map[string]*net.TCPConn)
	send_map_mutex := sync.RWMutex{}
	introduce_chan := make(chan string)
	gossip_chan := make(chan string)
	working_chan := make(chan bool)

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
			}
		}
	}

	localhost := local_ip_address + ":" + port_number
	connect_message := "CONNECT " + nodename + " " + local_ip_address + " " + port_number + "\n"
	connect_message_byte := []byte(connect_message)

	go start_server(port_number, localhost, send_map, send_map_mutex, gossip_chan, introduce_chan)
	go addRemote(nodename, local_ip_address, port_number, send_map, send_map_mutex, gossip_chan, introduce_chan)

	//Connect to server
	serverhost = server_address + ":" + server_portnumber
	for {
		tcp_add, _ := net.ResolveTCPAddr("tcp", serverhost)
		server_connection, err := net.DialTCP("tcp", nil, tcp_add)
		if err != nil {
			fmt.Println("#Failed to connect to the server")
			working_chan <- true
			break
		}

		defer server_connection.Close()
		server_connection.Write(connect_message_byte)
		go readMessage(port_number, server_connection, send_map, send_map_mutex, gossip_chan, introduce_chan)
		break
	}

	go gossip_transaction(localhost, send_map, send_map_mutex, gossip_chan)

	<-working_chan
	fmt.Println("Shall not reach here")
}

func signal_handler(){
	select {
		case <- cleanup_chan:
		case <- thanos_chan:
	}

	fmt.Println("Experment Ended, logging files")
	fmt.Println(len(holdback_transaction))
	if(write_to_file){
		file_name := "logs/" + program_start_time + ".csv"
		file, err := os.Create(file_name)
		if err != nil {
			fmt.Println("Error while creating file")
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		fmt.Println("created file")
		writer.Write([]string{"Port Number","Transaction ID", "Latency"})
		for port_num, xaction_list := range holdback_transaction{
			for _, xaction := range xaction_list{
				xaction_split := strings.Split(xaction, " ")
				writer.Write([]string{port_num, xaction_split[1], xaction_split[2]})
			}
		}
		writer.Flush()
	}
	global_working_chan <- true
}

func main(){

	cleanup_chan = make(chan os.Signal)
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	global_working_chan = make(chan bool)
	thanos_chan = make(chan bool)

	file_name := "logs/" + program_start_time + ".csv"
	os.Create(file_name)

	go signal_handler()

	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.MkdirAll("logs", os.ModePerm)
	} 

	if os.Args[len(os.Args)-1] == "silent" {
		write_to_file = false
	} else {
		write_to_file = true
	}

	program_start_time = time.Now().String()

	holdback_transaction = make(map[string][]string)

	if(os.Args[1] == "quick"){
		base_name := os.Args[2]
		base_port := os.Args[3]
		quantity_string := os.Args[4]
		quantity_integer, _ := strconv.Atoi(quantity_string)

		for i := 0; i < quantity_integer; i++ {
			node_name := base_name + strconv.Itoa(i)
			base_port_integer, _ := strconv.Atoi(base_port)
			port_number := strconv.Itoa(base_port_integer + i)

			holdback_transaction_mutex.Lock()
			holdback_transaction[port_number] = []string{}
			holdback_transaction_mutex.Unlock()
			time.Sleep(1*time.Second)
			go node(node_name, port_number)
		}
	} else {
		for i := 0; i < len(os.Args) / 2; i++ {
			node_name := os.Args[2*i+1]
			port_number := os.Args[2*i+2]
			holdback_transaction_mutex.Lock()
			holdback_transaction[port_number] = []string{}
			holdback_transaction_mutex.Unlock()
			time.Sleep(1*time.Second)
			go node(node_name, port_number)
		}
	}

	<-global_working_chan
	fmt.Println("Exiting..")
}