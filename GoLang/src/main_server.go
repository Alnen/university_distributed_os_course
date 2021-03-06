package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
	tsp_task_manager "tsp/task_manager"
)

type (
	MessageCmdType int
	ServerCmdType  int
	ServerState    int
	MessageType    struct {
		Conn *net.Conn
		Cmd  MessageCmdType
		Data []byte
	}
)

const (
	PRINT_DATA    MessageCmdType = iota
	GET_CALC_DATA                = iota
	PROC_ANSWER                  = iota
)

const (
	SERVER_START ServerCmdType = iota
	SERVER_QUIT                = iota
	ADD_NEW_TASK               = iota
)

const (
	RUNNING ServerState = iota
	STOPPED             = iota
)

var (
	new_client_id int
	new_worker_id int
	new_task_id   int
	logger        *log.Logger
	ch_cmd        chan ServerCmdType
	server_state  ServerState
)

func main() {
	tsp_task_manager.LogEnable = false
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "-l" {
			tsp_task_manager.LogEnable = true
			break
		}
	}
	var wg_server sync.WaitGroup
	new_client_id = 0
	new_worker_id = 0
	new_task_id = 0
	ch_cmd = make(chan ServerCmdType)
	server_state = STOPPED
	logger = log.New(os.Stdout, "SERVER: ", log.Ldate|log.Ltime)

	// start server
	wg_server.Add(1)
	go server_thread(&wg_server)

	var line string
	for {
		_, err := fmt.Scanln(&line)
		if err != nil {
			fmt.Printf("Input command error: %v", err)
			return
		}
		switch line {
		case "start":
			if server_state == STOPPED {
				wg_server.Add(1)
				go server_thread(&wg_server)
			} else {
				fmt.Println("SERVER COMMAND FAILED: Server is already running!")
			}
		case "stop":
			if server_state == RUNNING {
				ch_cmd <- SERVER_QUIT
				wg_server.Wait()
			} else {
				fmt.Println("SERVER COMMAND FAILED: Server is already stopped!")
			}
		case "quit":
			if server_state == RUNNING {
				ch_cmd <- SERVER_QUIT
				wg_server.Wait()
			}
			fmt.Println("QUIT")
			return
		}
	}
}

func server_thread(wg_server *sync.WaitGroup) {
	defer wg_server.Done()
	server_state = RUNNING
	logger.Println("Launching server...")
	runtime.GOMAXPROCS(runtime.NumCPU())
	tsp_task_manager.CreateTaskManager()

	// listen workers
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:5000")
	if nil != err {
		logger.Printf("ResolveTCPAddr (worker) error: %v\n", err)
		server_state = STOPPED
		return
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		logger.Printf("workers listen error: %v\n", err)
		server_state = STOPPED
		return
	}
	var wg_workers_accept sync.WaitGroup
	wg_workers_accept.Add(1)
	go accept_workers(listener, &wg_workers_accept)

	// listen clients
	laddr, err = net.ResolveTCPAddr("tcp", "127.0.0.1:6000")
	if nil != err {
		logger.Printf("ResolveTCPAddr (client) error: %v\n", err)
		server_state = STOPPED
		return
	}
	listener, err = net.ListenTCP("tcp", laddr)
	if err != nil {
		logger.Printf("clients listen error: %v\n", err)
		server_state = STOPPED
		return
	}
	var wg_clients_accept sync.WaitGroup
	wg_clients_accept.Add(1)
	go accept_clients(listener, &wg_clients_accept)
	logger.Println("Server work is started")

	// listen commands
	for {
		cmd := <-ch_cmd
		if cmd == SERVER_QUIT {
			// server quit
			server_state = STOPPED
			wg_workers_accept.Wait()
			wg_clients_accept.Wait()
			logger.Println("Server work is finished")
			return
		}
	}
}

func accept_clients(listener *net.TCPListener, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		if server_state == STOPPED {
			return
		}
		listener.SetDeadline(time.Now().Add(time.Duration(time.Second)))
		conn, err := listener.Accept()
		if err != nil {
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() && netErr.Temporary() {
				continue
			} else {
				logger.Printf("accept client error: %v\n", err)
				server_state = STOPPED
				return
			}
		}
		client := tsp_task_manager.ClientInfo{new_client_id, &conn}
		new_client_id++
		tsp_task_manager.AddNewClient(client)
		logger.Println("I'm accept client #", client.ID)
		go listen_client(client)
	}
}

func accept_workers(listener *net.TCPListener, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		if server_state == STOPPED {
			return
		}
		listener.SetDeadline(time.Now().Add(time.Duration(time.Second)))
		conn, err := listener.Accept()
		if err != nil {
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() && netErr.Temporary() {
				continue
			} else {
				logger.Printf("accept worker error: %v\n", err)
				server_state = STOPPED
				return
			}
		}
		worker := &tsp_task_manager.WorkerInfo{new_worker_id, &conn, -1}
		tsp_task_manager.AddNewWorker(worker)
		logger.Println("I'm accept worker #", new_worker_id)
		go listen_worker(worker)
		new_worker_id++
	}
}

func listen_client(client tsp_task_manager.ClientInfo) {
	for {
		var data_size int64
		err := binary.Read(*client.Conn, binary.LittleEndian, &data_size)
		if err != nil {
			if err == io.EOF {
				logger.Printf("Close client(%d) connection\n", client.ID)
			} else {
				logger.Printf("Reading data (client) error: %v", err)
			}
			return
		}
		data := make([]byte, data_size)
		_, err = (*client.Conn).Read(data)
		if err != nil {
			if err == io.EOF {
				logger.Printf("Close client(%d) connection\n", client.ID)
			} else {
				logger.Printf("Reading data (client) error: %v", err)
			}
			return
		}
		go tsp_task_manager.SolveTask(client, data, new_task_id)
		new_task_id++
	}
}

func listen_worker(worker *tsp_task_manager.WorkerInfo) {
	for {
		var data_size int64
		err := binary.Read(*worker.Conn, binary.LittleEndian, &data_size)
		if err != nil {
			logger.Printf("Reading data size (worker) error: %v", err)
			return
		}
		data := make([]byte, data_size)
		_, err = (*worker.Conn).Read(data)
		if err != nil {
			logger.Printf("Reading data (worker) error: %v", err)
			return
		}
		if string(data[0]) == "q" {
			return
		}
		// write answer
		tsp_task_manager.AnswerHandler((*worker).CurrentTask, data)
		tsp_task_manager.AddFreeWorker(worker)
	}
}
