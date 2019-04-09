package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
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
	gossip_fanout = 20
	history = 100
	serverhost string
)

var (
	working_chan chan bool
	introduce_chan chan string
	gossip_chan chan string
	cleanup_chan chan os.Signal
	start_time_time time.Time
)

var (
	send_map map[string]*net.TCPConn
	send_map_mutex = sync.RWMutex{}
	bandwidth_map map[string]int
	bandwidth_map_mutex = sync.RWMutex{}
	ip_2_index map[string]string
	send_history map[string][]string
	account map[int]int
)

var (
	holdback_transaction map[string]string
	current_log []string
	holdback_mutex = sync.Mutex{}
	pointer int
)

var (
	bandwidth_start_time time.Time
)

func checkErr(err error) int {
	if err != nil {
		if err.Error() == "EOF" {
			//fmt.Println(err)
			return 0
		}
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
			for _, value := range result {
				if temp == value {
					appear = true
				}
			}

			if appear == false {
				result = append(result, temp)
				break
			}
			temp += 1
		}
	}
	return result
}

func getCurrentDuration() string {
	duration_float := time.Since(start_time_time).Seconds()
	duration_string := fmt.Sprintf("%d", int(duration_float))
	return duration_string
}

func gossip_transaction(){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	for {
		gossip_message := <-gossip_chan
		b := []byte(gossip_message)

		send_map_mutex.RLock()
		skipped := 0
		for _, conn := range send_map {
			send_map_mutex.RUnlock()
			
			if (conn.RemoteAddr().String() == localhost) {
				send_map_mutex.RLock()
				skipped += 1
				continue
			}

			send_map_mutex.RLock()
			conn.Write(b)

		}
		bandwidth_map_mutex.Lock()
		bandwidth_map[getCurrentDuration()] += len(b) * (len(send_map) - skipped)
		bandwidth_map_mutex.Unlock()
		send_map_mutex.RUnlock()
	}
}

func periodically_send_transaction(){
	duration, _ := time.ParseDuration("10ms")
	time.Sleep(duration)

	total_len := 0
	for i:= 0; i < history; i++ {
		index := len(holdback_transaction) - (i+1)
		if index < 0 {
			break
		}
		send_message := []byte(current_log[index])
		send_message_key := strings.Split(current_log[index], " ")[2]

		count := 0
		send_map_mutex.Lock()
		for remote_host, conn := range send_map{
			send_map_mutex.Unlock()
			if count == gossip_fanout {
				send_map_mutex.Lock()
				break
			}

			found := false
			for _, remote := range send_history[send_message_key] {
				if remote_host == remote {
					found = true
					break
				}
			}

			if found {
				send_map_mutex.Lock()
				continue
			}

			send_history[send_message_key] = append(send_history[send_message_key], remote_host)

			conn.Write(send_message)
			total_len += len(send_message)

			count += 1
			send_map_mutex.Lock()
		}
		send_map_mutex.Unlock()
		bandwidth_map_mutex.Lock()
		bandwidth_map[getCurrentDuration()] += total_len
		bandwidth_map_mutex.Unlock()
	}
}

func printTransaction(port_num string, xaction string) string{
	xaction_split := strings.Split(xaction, " ")	
	time_string := xaction_split[1]
	time_float,_ := strconv.ParseFloat(time_string,64)
	currt := time.Now()
	current_float := float64(currt.UTC().UnixNano()) / 1000.0 / 1000.0 / 1000.0
	time_difference := current_float - time_float
	time_difference_string := fmt.Sprintf("%f", time_difference)
	return time_difference_string
}

func readMessage(node_name string, ip_address string, port_number string, conn *net.TCPConn){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	buff := make([]byte, 256)
	for {
		j, err := conn.Read(buff)
		flag := checkErr(err)
		if flag == 0 {
			failed_remote := conn.RemoteAddr().String()
			if(failed_remote == serverhost){
				port_prefix := ip_2_index[local_ip_address] + "_" + port_number
				fmt.Println(port_prefix + "Server failed, aborting...")
				working_chan <- true
			}
			send_map_mutex.Lock()
			delete(send_map, failed_remote)
			send_map_mutex.Unlock()
			break
		}
		bandwidth_map_mutex.Lock()
		bandwidth_map[getCurrentDuration()] += j
		bandwidth_map_mutex.Unlock()
		recevied_lines := strings.Split(string(buff[0:j]), "\n")
		for _, line := range recevied_lines {
			line_split := strings.Split(line, " ")
		
			if(line_split[0] == "INTRODUCE"){
				if(len(line_split) != 4){
					continue				
				}
				introduce_chan <- line
			}

			if(line_split[0] == "DIE" || line_split[0] == "QUIT"){
				working_chan <- true
			}

			if(line_split[0] == "TRANSACTION"){
				if(len(line_split) != 6){
					continue
				}
				holdback_mutex.Lock()
				found := false
				for key, _ := range holdback_transaction {
					if line_split[2] == key {
						found = true
						break
					}
				}

				if(found == true){
					holdback_mutex.Unlock()
					continue
				}

				time_difference := printTransaction(port_number, line)

				holdback_transaction[line_split[2]] = time_difference

				current_log = append(current_log, line)
				for i := len(current_log) - 1; i > 0; i-- {
					curr_spilt := strings.Split(current_log[i], " ")
					prev_split := strings.Split(current_log[i-1], " ")
					if(curr_spilt[1] < prev_split[1]){
						temp := current_log[i-1]
						current_log[i-1] = current_log[i]
						current_log[i] = temp
					}
				}

				holdback_mutex.Unlock()
				gossip_chan <- ("INTRODUCE " + node_name + " " + ip_address + " " + port_number + "\n" + line + "\n")
			}
		}
	}
}

func addRemote(node_name string, ip_address string, port_number string){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	for {
		line := <-introduce_chan
		line_split := strings.Split(line, " ")
		remotehost := line_split[2] + ":" + line_split[3]

		send_map_mutex.RLock()		
		if _, ok := send_map[remotehost]; ok {
			send_map_mutex.RUnlock()
			continue 
		}
		send_map_mutex.RUnlock()

		tcp_add, _ := net.ResolveTCPAddr("tcp", remotehost)
		remote_connection, err := net.DialTCP("tcp", nil, tcp_add)
		if err != nil {
			continue
		}

		if remote_connection == nil {
			continue		
		}
		
		send_map_mutex.Lock()
		send_map[remotehost] = remote_connection
		send_map_mutex.Unlock()

		defer remote_connection.Close()
		go readMessage(node_name, ip_address, port_number, remote_connection)
		send_map_mutex.RLock()
		sendMessageLen := 0
		for _, conn := range send_map{
			send_map_mutex.RUnlock()
			send_message := []byte("INTRODUCE " + node_name + " " + ip_address + " " + port_number + "\n" + line + "\n")
			sendMessageLen = len(send_message)
			conn.Write(send_message)
			send_map_mutex.RLock()
		}
		bandwidth_map_mutex.Lock()
		bandwidth_map[getCurrentDuration()] += sendMessageLen * len(send_map)
		bandwidth_map_mutex.Unlock()
		send_map_mutex.RUnlock()
	}
	close(introduce_chan)
}

func start_server(node_name string, ip_address string, port_num string){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)

	for {
		tcp_addr, _ := net.ResolveTCPAddr("tcp", localhost)
		tcp_listen, err := net.ListenTCP("tcp", tcp_addr)
			
		if err != nil {
			continue
		}

		fmt.Println("Start listening on ")
		// Accept Tcp connection from other VMs
		for {		
			conn, _ := tcp_listen.AcceptTCP()
			if conn == nil{
				continue
			}
			defer conn.Close()

			go readMessage(node_name, ip_address, port_num, conn)
		}

		break
	}
}

func global_map_init(){
	send_map = make(map[string]*net.TCPConn)
	bandwidth_map = make(map[string]int)
	ip_2_index = map[string]string{
		"10.192.137.227":"0",
		"172.22.158.52":"2",
		"172.22.94.61":"3",
		"172.22.156.53":"4",
		"172.22.158.53":"5",
		"172.22.94.62":"6",
		"172.22.156.54":"7",
		"172.22.158.54":"8",
		"172.22.94.63":"9",
		"172.22.156.55":"10",
	}
	send_history = make(map[string][]string)
	holdback_transaction = make(map[string]string)
}

func channel_init(){
	introduce_chan = make(chan string)
	gossip_chan = make(chan string)
	working_chan = make(chan bool)
	cleanup_chan = make(chan os.Signal)
}

func main_init(){
	global_map_init()
	channel_init()
}

func signal_handler(){
	<-cleanup_chan
	fmt.Println("signal_handler invoked")
	working_chan <- true
}

func get_local_ip_address(port_number string){
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

	localhost = local_ip_address + ":" + port_number
}

func main(){
	main_init()
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	if len(os.Args) != 4 {
		fmt.Println(os.Stderr, "Incorrect number of parameters")
		fmt.Println(len(os.Args))
		os.Exit(1)
	}

	node_name := os.Args[1]
	port_number := os.Args[2]

	start_time_int, _ := strconv.ParseInt(os.Args[3],10,64)
	start_time_time = time.Unix(start_time_int,0)

	get_local_ip_address(port_number)

	latency_filename := "latency/" + "latency" + ip_2_index[local_ip_address]+ "_" + port_number + ".csv"
	file, errf := os.Create(latency_filename)

	if errf != nil{
		fmt.Printf("port number %s failed to create its latency file \n", port_number)
	}
	
	bandwidth_file_name := "bandwidth/" + "bandwidth" + ip_2_index[local_ip_address]+ "_" + port_number + ".csv"
	bandwidth_file, errb:= os.Create(bandwidth_file_name)

	if errb != nil{
		fmt.Printf("port number %s failed to create the latency file \n", port_number)	
	}

	latencty_writer := csv.NewWriter(file)
	latencty_writer.Flush()

	bandwidth_writer := csv.NewWriter(bandwidth_file)
	bandwidth_writer.Flush()
	
	connect_message := "CONNECT " + node_name + " " + local_ip_address + " " + port_number + "\n"
	connect_message_byte := []byte(connect_message)

	go signal_handler()
	go start_server(node_name, local_ip_address, port_number)
	go addRemote(node_name, local_ip_address, port_number)

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
		go readMessage(node_name, local_ip_address, port_number, server_connection)
		break
	}

	//go self_introduction()
	go gossip_transaction()
	go periodically_send_transaction()
	<-working_chan
	time.Sleep(5*time.Second)
	latencty_writer_mutex := sync.Mutex{}
	latencty_writer_mutex.Lock()
	port_prefix := ip_2_index[local_ip_address] + "_" + port_number
	for transaction, time_difference := range holdback_transaction {
		latencty_writer.Write([]string{transaction,time_difference})
	}
	latencty_writer.Flush()
	latencty_writer_mutex.Unlock()

	bandwidth_writer_mutex := sync.Mutex{}
	bandwidth_writer_mutex.Lock()
	for time, bytes := range bandwidth_map {
		str_bytes := strconv.Itoa(bytes)
		bandwidth_writer.Write([]string{time,str_bytes})
	}	
	bandwidth_writer.Flush()
	bandwidth_writer_mutex.Unlock()
	fmt.Println(port_prefix + "Finished writing all files")
}

/*
	package:
	import "crypto/sha256"
	import "encoding/json"

	var:
	collect_logs []String

	struct:

	head_chain *block
	tail_chain *block

	type block struct{
		index int `json: "index"`
		prev_hash String `json: "prev_hash"`
		transaction_logs []String `json: "transaction_logs"`
		solution String `json: "solution"`
		next *block `json: "next"`
	}

	map:
	account map[int]int // ps: account 0 has infinite money, we just need to check other account

	Algorithm:
	1. Say, we can collect 100 transaction logs for each block (collect_logs) and check Consensus Rules at the same time (account).
		case 1: transfer from account always succeed.
		case 2: transfer from account n rather than 0, check account map to see if it has enough money, if so accept, otherwise reject.

	2. After collect enough logs, converting collect_log to []byte and use sha256.Sum256 to hash it
       then send the hash for this block to server for solving and waiting solution.

	3. While waiting for solution:
		case 1: if receiving block from whatever node, check the index of the received block,
				if the index <= tail.index, just ignore it
				if the index > tail.index and verify ok, add it to current chain and start a new term from 1, otherwise ignore it.

		// To use the longest-chain rule, I think we can make it simple, that is, each time we receive two acceptable block,
           we always choose the block with the smaller index. (that's my idea)
		case 2: if we are the node that first solves the puzzle, that we create a new block(block struct: hash, tail.index + 1, collect_logs, solution)
				and broadcast to other nodes through gossip function.

	How to broadcast a block struct through network:
	We can use json package, when we broadcast a block, first use json.Marshal it to []byte and then send it out.
	When we receive a block, we can use json.Unmarshal to decode it to the block struct and then do step 3: case 1.
 */