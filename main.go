package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
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

type Block struct {
	Index int `json: "index"`
	Prioirty int `json: "priority"`
	Created_time time.Time `json: "created_time"`
	Prev_hash string `json: "prev_hash"`
	Transaction_logs []string `json: "transaction_logs"`
	Solution string `json: "solution"`
	State map[int]int `json: "state"`
	Next *Block `json: "next"`
}

var (
	tail_chain = &Block{Index: 1}
	candidates []*Block
)

var (
	local_ip_address string
	localhost string
	port_prefix string
	port_priority int
)

var (
	//server_address = "10.192.137.227"
	server_address = "172.22.156.52"
	server_portnumber = "8888" //Port number listen on
	gossip_fanout = 20
	batch_size = 25
	history = 100
	has_issued_solve = false
	max_number_of_nodes_per_machine = 12
	serverhost string
)

var (
	working_chan chan bool
	introduce_chan chan string
	gossip_chan chan string
	cleanup_chan chan os.Signal
	solved_chan chan string
	start_time_time time.Time
)

var (
	send_map map[string]*net.TCPConn
	send_map_mutex = sync.RWMutex{}
	solutionMap map[string]*Block
	bandwidth_map map[string]int
	bandwidth_map_mutex = sync.RWMutex{}
	ip_2_index map[string]string
	send_history map[string][]string
	uncomitted_tail map[int]int
	split_map map[int]int
	split_map_mutex = sync.RWMutex{}
	block_latency_map map[string]string
	block_latency_map_mutex = sync.RWMutex{}
	split_time []string
)

var (
	logs_analysis map[string]string
	collect_logs []string
	tentative_blocks []*Block
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

func priorityLargerThan(attacker int, defenser int) bool {
	if attacker == port_priority {
		return true
	}

	if defenser == port_priority {
		return false
	}

	return attacker > defenser
}

func getCurrentDuration(precision string) string {
	duration_float := time.Since(start_time_time).Seconds()
	var duration_string string
	if precision == "int" {
		duration_string = fmt.Sprintf("%d", int(duration_float))
	} else {
		duration_string = fmt.Sprintf("%f", int(duration_float))
	}
	
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
		bandwidth_map[getCurrentDuration("int")] += len(b) * (len(send_map) - skipped)
		bandwidth_map_mutex.Unlock()
		send_map_mutex.RUnlock()
	}
}

func periodically_send_transaction(){
	duration, _ := time.ParseDuration("10ms")
	time.Sleep(duration)

	total_len := 0
	for i:= 0; i < history; i++ {
		index := len(logs_analysis) - (i+1)
		if index < 0 {
			break
		}
		send_message := []byte(collect_logs[index])
		send_message_key := strings.Split(collect_logs[index], " ")[2]

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
		bandwidth_map[getCurrentDuration("int")] += total_len
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
	buff := make([]byte, 10000)
	for {
		j, err := conn.Read(buff)
		flag := checkErr(err)
		if flag == 0 {
			failed_remote := conn.RemoteAddr().String()
			if(failed_remote == serverhost){

				fmt.Println(port_prefix + "Server failed, aborting...")
				working_chan <- true
			}
			send_map_mutex.Lock()
			delete(send_map, failed_remote)
			send_map_mutex.Unlock()
			break
		}

		bandwidth_map_mutex.Lock()
		bandwidth_map[getCurrentDuration("int")] += j
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

				source, _ := strconv.Atoi(line_split[3])
				destination, _ := strconv.Atoi(line_split[4])
				amount, _ := strconv.Atoi(line_split[5])

				_, ok := tail_chain.State[source]
				if(source != 0 && (!ok || tail_chain.State[source] < amount)) {
					continue
				}

				holdback_mutex.Lock()
				found := false
				for key, _ := range logs_analysis {
					if line_split[2] == key {
						found = true
						break
					}
				}

				if(found == true){
					holdback_mutex.Unlock()
					continue
				}

				printTransaction(port_number, line)

				logs_analysis[line_split[2]] = line_split[1]

				collect_logs = append(collect_logs, line)

				uncomitted_tail[source] -= amount
				uncomitted_tail[destination] += amount

				fmt.Println(len(collect_logs))
				for len(collect_logs) > batch_size {
					var new_tentative_block *Block

					if tail_chain.Index > 1 {
						new_tentative_block = &Block{Index: tail_chain.Index + 1, Prioirty: port_priority, Created_time: time.Now(), Prev_hash: tail_chain.Solution, Transaction_logs:collect_logs[:batch_size]}
					} else {
						new_tentative_block = &Block{Index: tail_chain.Index + 1, Prioirty: port_priority, Created_time: time.Now(), Transaction_logs:collect_logs[:batch_size]}
					}

					new_tentative_block.State = make(map[int]int)
					for account, amount := range uncomitted_tail {
						new_tentative_block.State[account] = amount
					}

					collect_logs = collect_logs[batch_size:]

					hash_string := ""
					if(new_tentative_block.Index > 1) {
						hash_string += new_tentative_block.Prev_hash
					}

					for _, value := range new_tentative_block.Transaction_logs {
						hash_string += value
					}

					sum := sha256.Sum256([]byte(hash_string))
					sum_string := fmt.Sprintf("%x", sum)
					solutionMap[sum_string] = new_tentative_block
					solved_chan <- sum_string
				}

				/*
				for i := len(collect_logs) - 1; i > 0; i-- {
					curr_spilt := strings.Split(collect_logs[i], " ")
					prev_split := strings.Split(collect_logs[i-1], " ")
					if(curr_spilt[1] < prev_split[1]){
						temp := collect_logs[i-1]
						collect_logs[i-1] = collect_logs[i]
						collect_logs[i] = temp
					}
				}
				*/
				holdback_mutex.Unlock()
				gossip_chan <- ("INTRODUCE " + node_name + " " + ip_address + " " + port_number + "\n" + line + "\n")
			}

			if(line_split[0] == "SOLVED"){
				fmt.Println("Received a SOLVED message ")
				if(len(line_split) != 3){
					continue
				}

				if prev_block, ok := solutionMap[line_split[1]]; ok {
					prev_block.Solution = line_split[2]
					fmt.Println("send_length = ", len(prev_block.State))
					/*multicast prev block*/

					/*
					type Block struct {
						index int `json: "index"`
						prioirty int `json: "priority"`
						prev_hash string `json: "prev_hash"`
						transaction_logs []string `json: "transaction_logs"`
						solution string `json: "solution"`
						state map[int]int `json: "state"`
						next *Block `json: "next"`
					}
					*/

					createdBlock := Block{Index: prev_block.Index, Prioirty:prev_block.Prioirty, Created_time:time.Now(), Prev_hash:prev_block.Prev_hash, Transaction_logs:prev_block.Transaction_logs, Solution:prev_block.Solution, State:prev_block.State, Next:prev_block.Next}

					/*Take down how much time does it take for each transaction to appear in a block*/
					for _, value := range createdBlock.State {

					}

					prevBlockBytes, err := json.Marshal(createdBlock)

					if err != nil {
						fmt.Println("Error while creating the new block", err)
					}

					prefix := "BLOCK "
					suffix := "\n"
					tail_chain = &createdBlock
					newBytes := []byte(prefix + string(prevBlockBytes) + suffix)


					send_map_mutex.RLock()
					for _, conn := range send_map {
						send_map_mutex.RUnlock()
						if(conn.RemoteAddr().String() == localhost){
							send_map_mutex.RLock()
							continue
						}

						conn.Write(newBytes)
						bandwidth_map_mutex.Lock()
						bandwidth_map[getCurrentDuration("int")] += len(newBytes)
						bandwidth_map_mutex.Unlock()
						send_map_mutex.RLock()

					}
					send_map_mutex.RUnlock()

					/*not sure whether I need to empty the solution list now*/
				}
			}

			if(line_split[0] == "BLOCK"){

				/*
				func ParseDuration string 2 duration

				 */
				var received_block Block
				err := json.Unmarshal([]byte(line[6:]), &received_block)
				if err != nil {
					fmt.Println(err)
				}

				if received_block.Index == tail_chain.Index + 1 || (received_block.Index == tail_chain.Index && priorityLargerThan(received_block.Index, tail_chain.Index)){
					if received_block.Index != tail_chain.Index + 1 {
						fmt.Println("Split occured!")
						split_map_mutex.Lock()
						split_map[tail_chain.Index] += 1
						split_map_mutex.Unlock()
						split_time = append(split_time, getCurrentDuration("float"))
					}
					
					duration := time.Since(received_block.Created_time)
					durationString := fmt.Sprintf("%s", duration)

					priorityString := strconv.Itoa(received_block.Prioirty)
					block_latency_map[priorityString] = durationString[:len(durationString)-2]

					fmt.Println("received_block_length = ", len(received_block.State))
					tail_chain.Next = &received_block
					tail_chain = &received_block
					uncomitted_tail = received_block.State
					for key, _ := range(solutionMap){
						delete(solutionMap, key)
					}
					collect_logs = nil
				}
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
		for _, conn := range send_map {
			send_map_mutex.RUnlock()
			send_message := []byte("INTRODUCE " + node_name + " " + ip_address + " " + port_number + "\n" + line + "\n")
			sendMessageLen = len(send_message)
			conn.Write(send_message)
			send_map_mutex.RLock()
		}
		bandwidth_map_mutex.Lock()
		bandwidth_map[getCurrentDuration("int")] += sendMessageLen * len(send_map)
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

func request_solution(server_connection *net.TCPConn){

	for {
		sum_string := <- solved_chan
		send_string := "SOLVE " + sum_string + "\n"
		fmt.Println("send_string = ", send_string)
		server_connection.Write([]byte(send_string))
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
	logs_analysis = make(map[string]string)
	block_latency_map = make(map[string]string)
	solutionMap = make(map[string]*Block)
	uncomitted_tail = make(map[int]int)
	split_map = make(map[int]int)
}

func channel_init(){
	introduce_chan = make(chan string)
	gossip_chan = make(chan string)
	working_chan = make(chan bool)
	solved_chan = make(chan string)
	cleanup_chan = make(chan os.Signal)
}

func block_init(){
	tail_chain.State = make(map[int]int)
	tail_chain.Prioirty = port_priority
	tail_chain.Created_time = time.Now()
}

func variable_init(port_number string){
	port_prefix = ip_2_index[local_ip_address] + "_" + port_number
	port_number_int, _ := strconv.Atoi(port_number)
	machine_number_int, _ := strconv.Atoi(port_number)
	port_priority = port_number_int * max_number_of_nodes_per_machine + machine_number_int
}

func main_init(port_number string){
	global_map_init()
	channel_init()
	variable_init(port_number)
	block_init()
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


	if len(os.Args) != 4 {
		fmt.Println(os.Stderr, "Incorrect number of parameters")
		fmt.Println(len(os.Args))
		os.Exit(1)
	}

	node_name := os.Args[1]
	port_number := os.Args[2]
	main_init(port_number)
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	
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
		fmt.Printf("port number %s failed to create the bandwidth file \n", port_number)
	}

	balance_file_name := "balance/" + "balance" + ip_2_index[local_ip_address]+ "_" + port_number + ".csv"
	balance_file, erra:= os.Create(balance_file_name)

	if erra != nil{
		fmt.Printf("port number %s failed to create the balance file \n", port_number)
	}

	blocklatency_file_name := "blocklatency/" + "blocklatency" + ip_2_index[local_ip_address]+ "_" + port_number + ".csv"
	blocklatency_file, errbl:= os.Create(blocklatency_file_name)

	if errbl != nil{
		fmt.Printf("port number %s failed to create the blocklatency file \n", port_number)
	}

	split_file_name := "split/" + "split" + ip_2_index[local_ip_address]+ "_" + port_number + ".csv"
	split_file, errs:= os.Create(split_file_name)

	if errs != nil {
		fmt.Printf("port number %s failed to create the split file \n", port_number)
	}

	latencty_writer := csv.NewWriter(file)
	latencty_writer.Flush()

	bandwidth_writer := csv.NewWriter(bandwidth_file)
	bandwidth_writer.Flush()

	balance_writer := csv.NewWriter(balance_file)
	balance_writer.Flush()

	blocklatency_writer := csv.NewWriter(blocklatency_file)
	blocklatency_writer.Flush()

	split_writer := csv.NewWriter(split_file)
	split_writer.Flush()

	connect_message := "CONNECT " + node_name + " " + local_ip_address + " " + port_number + "\n"
	connect_message_byte := []byte(connect_message)

	go signal_handler()
	go start_server(node_name, local_ip_address, port_number)
	go addRemote(node_name, local_ip_address, port_number)

	//Connect to server
	port_prefix := ip_2_index[local_ip_address] + "_" + port_number
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
		go request_solution(server_connection)
		go readMessage(node_name, local_ip_address, port_number, server_connection)
		break
	}

	//go self_introduction()
	go gossip_transaction()
	go periodically_send_transaction()

	<-working_chan
	time.Sleep(7*time.Second)

	latencty_writer_mutex := sync.Mutex{}
	latencty_writer_mutex.Lock()
	for _, value := range collect_logs {
		latencty_writer.Write([]string{port_number, value})
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

	balance_writer_mutex := sync.Mutex{}
	balance_writer_mutex.Lock()
	for account_int, value_int := range tail_chain.State {
		account_stirng := strconv.Itoa(account_int)
		value_string := strconv.Itoa(value_int)
		balance_writer.Write([]string{port_prefix, account_stirng, value_string})
	}

	balance_writer.Flush()
	balance_writer_mutex.Unlock()

	blocklatency_writer_mutex := sync.Mutex{}
	blocklatency_writer_mutex.Lock()
	for block_priority, latency := range block_latency_map {
		blocklatency_writer.Write([]string{port_prefix, block_priority, latency})
	}
	blocklatency_writer.Flush()
	blocklatency_writer_mutex.Unlock()

	split_writer_mutex := sync.Mutex{}
	split_writer_mutex.Lock()
	for index_int, count_int := range split_map{
		index_string := strconv.Itoa(index_int)
		count_string := strconv.Itoa(count_int)
		split_writer.Write([]string{index_string, count_string})
	}
	split_writer.Flush()
	split_writer_mutex.Unlock()

	fmt.Println(port_prefix + "finished writing all files")
}

/*
	package:
	import "crypto/sha256"
	import "encoding/json"

	var:
	logs_analysis []String

	struct:

	head_chain *block
	tail_chain *block

	type block struct{
		index int `json: "index"`
		prev_hash String `json: "prev_hash"`
		transaction_logs []String `json: "transaction_logs"`
		solution String `json: "solution"`
		next []*block `json: "next"`
	}

	map:
	account map[int]int // ps: account 0 has infinite money, we just need to check other account

	Algorithm:
	1. Say, we can collect 100 transaction logs for each block (logs_analysis) and check Consensus Rules at the same time (account).
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
		case 2: if we are the node that first solves the puzzle, that we create a new block(block struct: hash, tail.index + 1, logs_analysis, solution)
				and broadcast to other nodes through gossip function.

	How to broadcast a block struct through network:
	We can use json package, when we broadcast a block, first use json.Marshal it to []byte and then send it out.
	When we receive a block, we can use json.Unmarshal to decode it to the block struct and then do step 3: case 1.
 */