package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"fmt"
	"strconv"
	"io"
	"sync"
	tsp_types "tsp/types"
	tsp_solver "tsp/solver"
)

type WorkerCmdType int

const (
	CMD_SOLVE_NEW_TASK WorkerCmdType = iota
	CMD_QUIT                         = iota
)

var (
	ch_logs chan string
	ch_cmd    chan WorkerCmdType
	curr_task string
)

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

func listen_server(conn *net.Conn) {
	for {
		bufr := bufio.NewReader(*conn)
		cmd, err := bufr.ReadString('\000')
		if err != nil {
			str := fmt.Sprintf("reading error: %v", err)
			ch_logs <- str
			return
		}
		switch cmd {
		case "QUIT":
			ch_cmd <- CMD_QUIT
			return
		default:
			curr_task = cmd
			ch_cmd <- CMD_SOLVE_NEW_TASK
		}
	}
}

func start_worker() {
	destination := "127.0.0.1:5000"
	str := fmt.Sprintf("worker: connect to %s", destination)
	ch_logs <- str
	conn, err := net.Dial("tcp", destination)
	if err != nil {
		str = fmt.Sprintf("dial error: %v", err)
		ch_logs <- str
	}
	go listen_server(&conn)
	for {
		cmd := <-ch_cmd
		switch cmd {
		case CMD_QUIT:
			return
		case CMD_SOLVE_NEW_TASK:
			//read task
			task := tsp_types.TaskType{}
			task.FromString(curr_task)
			//do it
			answer := tsp_solver.SolveImpl(task)
			conn.Write([]byte(answer.ToString()+"\000"))
			// DO NEW TASK (curr task)
			// conn.Write([]byte("Task Answer\000"))
		}
	}
	defer conn.Close()
}

func main() {
	if len(os.Args) > 1 {
		worker_count, err := strconv.Atoi(os.Args[1]);
		if err != nil {
			fmt.Printf("Command Arg Error: %v\n", err)
		}
		fmt.Println("I'm try")
		var wg_logger sync.WaitGroup
		ch_logs = make(chan string, 100)
		ch_stop_logging := make(chan bool)
		wg_logger.Add(1)
		go log_thread(os.Stdout, ch_stop_logging, &wg_logger)
		for i:= 0; i < worker_count; i++ {
			go start_worker();
		}
		var line string
		for {
			_, err := fmt.Scanln(&line)
			if err != nil {
				fmt.Printf("Input command error: %v", err)
				return
			}
			switch line {
			case "quit":
				ch_cmd <- CMD_QUIT
				ch_stop_logging <- true
				wg_logger.Wait()
				fmt.Println("QUIT")
				return
			}
		}
	}
}
