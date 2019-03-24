package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"math/rand"
	"os/signal"
	"syscall"
	"strconv"
	"encoding/csv"
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
	ip_2_index map[string]string
)

var (
	holdback_transaction []string
	holdback_transaction_print []string
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

func getCurrentDuration() string {
	duration_float := time.Since(start_time_time).Seconds()
	duration_string := fmt.Sprintf("%f", duration_float)
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
		bandwidth_map[getCurrentDuration()] += len(b) * (len(send_map) - skipped)
		send_map_mutex.RUnlock()
	}
}

func periodically_send_transaction(){
	duration, _ := time.ParseDuration("1ms")
	time.Sleep(duration)	
        count := 0
        send_map_mutex.RLock()
        receivers := generateRandom(len(send_map) , gossip_fanout)
	send_map_mutex.RUnlock()
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

		total_len := 0
		for i := 0; i < history; i++ {
			index := len(holdback_transaction) - (i+1)
			if index < 0 {
				break
			}
			send_message := []byte(holdback_transaction[index])
			total_len += len(send_message)
			conn.Write(send_message)
		}

		bandwidth_map[getCurrentDuration()] += total_len
		send_map_mutex.RLock()
	}
	send_map_mutex.RUnlock()
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
				fmt.Println("Server failed, aborting...")
				working_chan <- true
			}
			send_map_mutex.Lock()
			delete(send_map, failed_remote)
			send_map_mutex.Unlock()
			break
		}

		bandwidth_map[getCurrentDuration()] += j
		
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
				//fmt.Printf("%s received a message from %s\n",port_number, conn.RemoteAddr().String())
				time_difference := printTransaction(port_number, line)
				holdback_transaction = append(holdback_transaction, line)
				holdback_transaction_print = append(holdback_transaction_print, line + " " + time_difference)
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
		bandwidth_map[getCurrentDuration()] += sendMessageLen * len(send_map)
		send_map_mutex.RUnlock()
	}
	close(introduce_chan)
}

func start_server(node_name string, ip_address string, port_num string) {
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
	fmt.Println("latnecy = ", latency_filename)
	file, errf := os.Create(latency_filename)

	if errf != nil{
		fmt.Printf("port number %s failed to open latency file \n", port_number)
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
	latencty_writer_mutex := sync.Mutex{}
	fmt.Printf("")
	fmt.Printf("port_number = %s, holdback_queue_length = %d, send_map_length = %d\n",port_number, len(holdback_transaction_print), len(send_map))
	latencty_writer_mutex.Lock()
	port_prefix := port_number+"_"+ip_2_index[local_ip_address]
	for _, transaction := range holdback_transaction_print {
		transaction_split := strings.Split(transaction, " ")
		latencty_writer.Write([]string{port_prefix,transaction_split[2],transaction_split[6]})
	}
	latencty_writer.Flush()
	latencty_writer_mutex.Unlock()
	fmt.Println("Finished writing all files")

	bandwidth_writer_mutex := sync.Mutex{}
	bandwidth_writer_mutex.Lock()
	for time, bytes := range bandwidth_map{
		str_bytes := strconv.Itoa(bytes)
		bandwidth_writer.Write([]string{port_prefix,time,str_bytes})
	}	
	bandwidth_writer.Flush()
	bandwidth_writer_mutex.Unlock()
}
