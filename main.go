package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Wrap struct {
	ip_address  string
	sender_name string
	vector      []int
	message     string
}

var (
	num_of_participants int64
	count               int64
	localhost           string
	local_ip_address    string
	port_number         string
	own_name            string
)

var (
	read_map         map[string]*net.TCPConn
	send_map         map[string]*net.TCPConn
	remote_ip_2_name map[string]string
	conn_2_port_num  map[*net.TCPConn]string
	ip_2_vectorindex map[string]int
)

var (
	vm_addresses = []string{"172.22.156.52:4444", "172.22.158.52:4444", "172.22.94.61:4444", "172.22.156.53:4444", "172.22.158.53:4444",
		"172.22.94.62:4444", "172.22.156.54:4444", "172.22.158.54:4444", "172.22.94.63:4444", "172.22.156.55:4444"}
	vector         = []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	start_chan     chan bool
	ready_chan     chan bool
	holdback_queue = []Wrap{}
)

var (
	connect_chan chan bool
)

var (
	server_address = "0.0.0.0"
	server_portnumber = "8888"
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

func printReady(ready chan bool) {
	for {
		if <-ready_chan {
			count--
		}
		if count == 0 {
			break
		}
	}
	fmt.Println("Ready")
}

func serialize(vec []int) string {
	result := "["
	for i := 0; i < len(vec); i++ {
		result += ","
		result += strconv.Itoa(vec[i])
	}
	result += ",]"
	return result
}

func deserialize(str string) []int {
	parse := str[2 : len(str)-2]
	s := strings.Split((parse), ",")
	var result []int
	for i := 0; i < len(vector); i++ {
		temp, _ := strconv.Atoi(s[i])
		result = append(result, temp)
	}
	return result
}

func deliver(update_index int, deliver_string string) {
	vector[update_index] += 1
	fmt.Println(deliver_string)
}

func able_to_deliver(received_vector []int, source_index int) bool {
	result := true
	for i := 0; i < len(vector); i++ {
		if i == source_index {
			if received_vector[i] != vector[i]+1 {
				return false
			}
		} else {
			if received_vector[i] > vector[i] {
				return false
			}
		}
	}
	return result
}

func readMessage(conn *net.TCPConn) {

	buff := make([]byte, 256)
	for {
		j, err := conn.Read(buff)
		//Check user leave or error happen
		flag := checkErr(err)
		if flag == 0 {
			s := strings.Split(conn.RemoteAddr().String(), ":")
			remote_ip := s[0]
			fmt.Println(remote_ip_2_name[remote_ip] + " has left")
			break
		}

		recevied_string_spilt := strings.Split(string(buff[0:j]), ";")

		if recevied_string_spilt[0] == "NAME" {
			fmt.Println("#" + recevied_string_spilt[1] + " (Address: " + conn.RemoteAddr().String() + ")" + " has joined the chat")
			remote_ip_2_name[strings.Split(conn.RemoteAddr().String(), ":")[0]] = recevied_string_spilt[1]
			ready_chan <- true
			continue
		}

		received_vector := deserialize(recevied_string_spilt[0])
		received_ip_address := recevied_string_spilt[1]
		received_name := recevied_string_spilt[2]
		received_message := recevied_string_spilt[3]
		deliver_string := received_name + ": " + received_message
		//fmt.Printf("#Current timestamp in this VM is: %v\n", vector)

		//Check whether it is able to deliver according to casual deliver order
		if able := able_to_deliver(received_vector, ip_2_vectorindex[received_ip_address]); able {
			deliver(ip_2_vectorindex[received_ip_address], deliver_string)
			//fmt.Printf("#After deliver, timestamp at this VM becomes: %v\n", vector)
			for {
				again := false
				for it := 0; it < len(holdback_queue); it++ {
					object := holdback_queue[it]
					if flag := able_to_deliver(object.vector, ip_2_vectorindex[object.ip_address]); flag {
						deliver(ip_2_vectorindex[object.ip_address], object.message)
						again = true
						break
					}
				}
				if again == false {
					break
				}
			}
		} else {
			fmt.Println("#Unable to Deliver the Message, put it to the hold back queue")
			Temp := Wrap{received_ip_address, received_name, received_vector, received_message}
			holdback_queue = append(holdback_queue, Temp)
			fmt.Printf("#Check holdback queue: %v\n", holdback_queue)
		}
	}
}

func multicast(name string) {
	for {
		var msg []byte
		var send_string string

		in := bufio.NewReader(os.Stdin)
		msg, _, _ = in.ReadLine()

		//Print info for debug
		//fmt.Println("#The message that you are about to send is: ", string(msg))
		vector[ip_2_vectorindex[local_ip_address]] += 1
		//fmt.Println("#The index of the timestamp at this VM is: ", ip_2_vectorindex[local_ip_address])
		//fmt.Printf("#After increament, timestamp at this VM becomes: %v\n", vector)

		//Format each piece of message before send it out
		send_vector := serialize(vector)
		send_string = send_vector + ";" + local_ip_address + ";" + name + ";" + string(msg)
		b := []byte(send_string)

		for _, conn := range send_map {
			if conn.RemoteAddr().String() == localhost {
				continue
			}
			conn.Write(b)
		}

	}
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
		conn_2_port_num[conn] = port_num
		read_map[conn.RemoteAddr().String()] = conn
		go readMessage(conn)
	}
}

func start_client(num_of_participants int64, name []byte) {

	//Create TCP connection to other VMs
	for i := int64(0); i < 10; i++ {
		if vm_addresses[i] != localhost {
			tcp_add, _ := net.ResolveTCPAddr("tcp", vm_addresses[i])
			conn, err := net.DialTCP("tcp", nil, tcp_add)
			if err != nil {
				//fmt.Println("#Service unavailable on this VM, try next VM" + vm_addresses[i])
				continue
			}
			defer conn.Close()

			conn.Write(name)
			send_map[conn.RemoteAddr().String()] = conn
		}
	}
	ready_chan <- true
	<-start_chan
}

func connect(message []byte){
	for {
		tcp_add, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:8888")
		conn, err := net.DialTCP("tcp", nil, tcp_add)
		if err != nil {
			fmt.Println("#Failed to connect to the server")
			continue
		}

		defer conn.Close()
		conn.Write(message)

		break
	}

	connect_chan <- true
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
	go connect(connect_message_byte)
	<-connect_chan

	fmt.Println("Finished sending connection")
}
