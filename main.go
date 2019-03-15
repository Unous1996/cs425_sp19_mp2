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
	gossip_fanout = 10
	serverhost string
)

var (
	working_chan chan bool
	introduce_chan chan string
	gossip_chan chan string
	cleanup_chan chan os.Signal
	program_start_time time.Time
)

var (
	send_map map[string]*net.TCPConn
	bandwidth_map map[string]int
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
				send_map_mutex.RLock()
				break
			}
			if (found == false  || conn.RemoteAddr().String() == localhost) {
				send_map_mutex.RLock()
				continue
			}
			bandwidth_map[time.Since(program_start_time).String()] += len(b) 
			send_map_mutex.RLock()
		}
		send_map_mutex.RUnlock()
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
	return_string := port_num + " " + xaction_split[2] + " " + time_difference_string
	fmt.Println(return_string)
	return return_string 
}

func readMessage(port_number string, conn *net.TCPConn){
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	buff := make([]byte, 256)
	for {
		j, err := conn.Read(buff)
		flag := checkErr(err)
		if flag == 0 {
			failed_remote := conn.RemoteAddr().String()
			if(failed_remote == serverhost){
				working_chan <- true
			}
			/*
			if(len(strings.Split(failed_remote,":")[1]) != 5){
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

				holdback_message := printTransaction(port_number, line)
				holdback_transaction = append(holdback_transaction, holdback_message)
				gossip_chan <- line
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
			fmt.Println("Failed while dialing the remote node " + remotehost)
		}

		if remote_connection == nil {
			continue		
		}
		
		send_map_mutex.Lock()		
		send_map[remotehost] = remote_connection
		send_map_mutex.Unlock()

		defer remote_connection.Close()
		go readMessage(port_number, remote_connection)

		send_map_mutex.RLock()
		for _, conn := range send_map{
			send_map_mutex.RUnlock()
			send_message := []byte("INTRODUCE " + node_name + " " + ip_address + " " + port_number + "\n" + line + "\n")
			bandwidth_map[time.Since(program_start_time).String()] =  len(send_message)
			conn.Write(send_message)
			send_map_mutex.RLock()
		}
		send_map_mutex.RUnlock()
	}
	close(introduce_chan)
}

func self_introduction(){
	time.Sleep(1*time.Second)
	send_map_mutex.RLock()
	for _, conn := range send_map{
		for  remotehost, _ := range send_map{
			ip_address := strings.Split(remotehost, ":")[0]
 			port_number := strings.Split(remotehost, ":")[1]
			send_message := []byte("INTRODUCE " + "node_name" + " " + ip_address + " " + port_number + "\n") 
			conn.Write(send_message)
		}	
	}
	send_map_mutex.RUnlock()
}


func start_server(port_num string) {
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

		go readMessage(port_num, conn)
	}
}

func global_map_init(){
	send_map = make(map[string]*net.TCPConn)
	bandwidth_map = make(map[string]int)
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

func main(){
	program_start_time = time.Now()
	main_init()
	signal.Notify(cleanup_chan, os.Interrupt, syscall.SIGTERM)
	if len(os.Args) != 4 {
		fmt.Println(os.Stderr, "Incorrect number of parameters")
		fmt.Println(len(os.Args))
		os.Exit(1)
	}

	self_nodename := os.Args[1]
	port_number := os.Args[2]
	//Get local ip address
	
	file_name := "logs/" + os.Args[3] + "/latency/" + port_number + ".csv" 
	file, errf := os.Create(file_name)
	if errf != nil{
		panic("error while creating latency file")	
	}
	bandwidth_file_name := "logs/" + os.Args[3] + "/bandwidth/" + port_number + ".csv"
	bandwidth_file, errb:= os.Create(bandwidth_file_name)

	if errb != nil{
		panic("error while creating bandwidth file")	
	}

	writer := csv.NewWriter(file)
	writer.Write([]string{"Port Number","Transaction ID", "Latency"})
	writer.Flush()

	bandwidth_writer := csv.NewWriter(bandwidth_file)
	bandwidth_writer.Write([]string{"Port Number", "Time Since Start", "Bandwidth"})
	
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
	connect_message := "CONNECT " + self_nodename + " " + local_ip_address + " " + port_number + "\n"
	connect_message_byte := []byte(connect_message)

	go signal_handler()
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

	go self_introduction()
	go gossip_transaction()

	<-working_chan
	fmt.Printf("port_number = %d, send_map_length = %d\n",port_number,len(send_map))
	for _, transaction := range holdback_transaction {
		transaction_split := strings.Split(transaction, " ")
		writer.Write([]string{port_number,transaction_split[1],transaction_split[2]})
	}
	writer.Flush()

	for time, bytes := range bandwidth_map{
		str_bytes := strconv.Itoa(bytes)
		bandwidth_writer.Write([]string{port_number,time,str_bytes})
	}	
	bandwidth_writer.Flush()
}
