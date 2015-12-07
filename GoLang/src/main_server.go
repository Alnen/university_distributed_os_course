package main

import (
	//tsp_solver "tsp_service/tsp/solver"
	"bufio"
	"container/list"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
	tsp_task_manager "tsp/task_manager"
	tsp_types "tsp/types"
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
	new_client_id           int
	new_worker_id           int
	input_global_task_queue chan *tsp_types.TaskType
	ch_logs                 chan string
	ch_input_messages       chan MessageType
	ch_cmd                  chan ServerCmdType
	server_state            ServerState
	worker_count            int
)

func main() {
	var wg_server, wg_logger sync.WaitGroup
	worker_count = 2
	new_client_id = 0
	new_worker_id = 0
	ch_cmd = make(chan ServerCmdType)
	ch_logs = make(chan string, 100)
	server_state = STOPPED
	ch_stop_logging := make(chan bool)

	// start server
	wg_server.Add(1)
	go server_thread(&wg_server)
	wg_logger.Add(1)
	go log_thread(os.Stdout, ch_stop_logging, &wg_logger)

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
				wg_logger.Add(1)
				go log_thread(os.Stdout, ch_stop_logging, &wg_logger)
			} else {
				fmt.Println("SERVER COMMAND FAILED: Server is already running!")
			}
		case "stop":
			if server_state == RUNNING {
				ch_cmd <- SERVER_QUIT
				ch_stop_logging <- true
				wg_server.Wait()
				wg_logger.Wait()
			} else {
				fmt.Println("SERVER COMMAND FAILED: Server is already stopped!")
			}
		case "quit":
			if server_state == RUNNING {
				ch_cmd <- SERVER_QUIT
				ch_stop_logging <- true
				wg_server.Wait()
				wg_logger.Wait()
			}
			fmt.Println("QUIT")
			return
		}
	}
}

func log_thread(writerHandle io.Writer, ch_stop_logging chan bool, wg_logger *sync.WaitGroup) {
	defer wg_logger.Done()
	logger := log.New(writerHandle,
		"SERVER: ",
		log.Ldate|log.Ltime)
	for {
		select {
		case msg := <-ch_logs:
			logger.Println(msg)
		case <-ch_stop_logging:
			return
		}
	}
}

func server_thread(wg_server *sync.WaitGroup) {
	defer wg_server.Done()
	server_state = RUNNING
	ch_logs <- "Launching server..."
	runtime.GOMAXPROCS(runtime.NumCPU())
	ch_input_messages = make(chan MessageType, worker_count)
	tsp_task_manager.CreateTaskManager()

	// listen workers
	laddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:5000")
	if nil != err {
		str := fmt.Sprintf("ResolveTCPAddr error: %v\n", err)
		ch_logs <- str
		server_state = STOPPED
		return
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		str := fmt.Sprintf("workers listen error: %v\n", err)
		ch_logs <- str
		server_state = STOPPED
		return
	}
	workers_list := list.New()
	var wg_workers_accept sync.WaitGroup
	wg_workers_accept.Add(1)
	go accept_workers(workers_list, listener, &wg_workers_accept)

	// listen clients
	laddr, err = net.ResolveTCPAddr("tcp", "127.0.0.1:6000")
	if nil != err {
		str := fmt.Sprintf("ResolveTCPAddr error: %v\n", err)
		ch_logs <- str
		server_state = STOPPED
		return
	}
	listener, err = net.ListenTCP("tcp", laddr)
	if err != nil {
		str := fmt.Sprintf("clients listen error: %v\n", err)
		ch_logs <- str
		server_state = STOPPED
		return
	}
	clients_list := list.New()
	var wg_clients_accept sync.WaitGroup
	wg_clients_accept.Add(1)
	go accept_clients(clients_list, listener, &wg_clients_accept)

	ch_logs <- "Server work is started"
	// listen commands
	for {
		cmd := <-ch_cmd
		if cmd == SERVER_QUIT {
			// server quit
			server_state = STOPPED
			wg_workers_accept.Wait()
			wg_clients_accept.Wait()
			ch_logs <- "Server work is finished"
			return
		}
	}
}

func accept_clients(clients_list *list.List, listener *net.TCPListener, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		if server_state == STOPPED {
			return
		}
		//fmt.Println("Server accept connections ...")
		listener.SetDeadline(time.Now().Add(time.Duration(time.Second)))
		conn, err := listener.Accept()
		if err != nil {
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() && netErr.Temporary() {
				continue
			} else {
				str := fmt.Sprintf("accept error: %v\n", err)
				ch_logs <- str
				server_state = STOPPED
				return
			}
		}
		client := tsp_task_manager.ClientInfo{new_client_id, &conn}
		new_client_id++
		clients_list.PushBack(client)
		ch_logs <- ("I'm accept client #" + strconv.Itoa(client.ID))
		go listen_client(client)
	}
}

func accept_workers(workers_list *list.List, listener *net.TCPListener, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		if server_state == STOPPED {
			return
		}
		//fmt.Println("Server accept connections ...")
		listener.SetDeadline(time.Now().Add(time.Duration(time.Second)))
		conn, err := listener.Accept()
		if err != nil {
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() && netErr.Temporary() {
				continue
			} else {
				str := fmt.Sprintf("accept error: %v\n", err)
				ch_logs <- str
				server_state = STOPPED
				return
			}
		}
		worker := tsp_task_manager.WorkerInfo{new_worker_id, &conn, nil}
		workers_list.PushBack(worker)
		tsp_task_manager.AddFreeWorker(worker)
		ch_logs <- ("I'm accept worker #" + strconv.Itoa(new_worker_id))
		go listen_worker(worker)
		new_worker_id++
	}
}

//func listen_client(client tsp_task_manager.ClientInfo, worker_list *list.List) {
func listen_client(client tsp_task_manager.ClientInfo) {
	for {
		line := make([]byte, 4096)
		actual_size, err := (*client.Conn).Read(line)
		if err != nil {
			str := fmt.Sprintf("Reading data error: %v", err)
			ch_logs <- str
			return
		}
		actual_line := line[:actual_size]
		//ch_logs <- ("listen_client: receive task " + string(actual_line))
		go tsp_task_manager.SolveTask(client, actual_line, ch_logs)
		/*
			if string(line) == "quit" {
				for e := worker_list.Front(); e != nil; e = e.Next() {
					worker := e.Value.(WorkerInfo)
					if client.ID == worker.CurrentTask.Client.ID {
						//worker.CurrentTask.Client = Conn
					}
				}
			} else {
				tsp_task_manager.AddNewTask(client, line)
			}
		*/
	}
}

func listen_worker(worker tsp_task_manager.WorkerInfo) {
	for {
		bufr := bufio.NewReader(*worker.Conn)
		line, err := bufr.ReadSlice('\000')
		if err != nil {
			str := fmt.Sprintf("Reading data error: %v", err)
			ch_logs <- str
			return
		}
		//ch_logs <- ("listen_worker: receive answer" + string(line))
		// write answer
		tsp_task_manager.ProcessingAnswer(line)
		tsp_task_manager.AddFreeWorker(worker)
		//(*worker.CurrentTask.Client.Conn).Write(line)
		/*
			if string(line) == "quit" {
				tsp_task_manager.tasks_queue <- worker.CurrentTask
			} else {
				worker.CurrentTask.Client.Conn.Write([]byte("Connection is accepted\000"))
			}
		*/
	}
}
