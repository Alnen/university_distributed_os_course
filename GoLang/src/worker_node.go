package main

import (
	//"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	tsp_solver "tsp_test/tsp/solver"
	tsp_types "tsp_test/tsp/types"
)

type WorkerCmdType int

var (
	ch_logs chan string
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

var answer_count int = 0

func print_matrix(task *tsp_types.TaskType) {
	fmt.Println("---------- MATRIX ---------")
	for i := 0; i < task.Size; i++ {
		for j := 0; j < task.Size; j++ {
			fmt.Printf("%11d",int((*task.Matrix)[i*task.Size+j]))
		}
		fmt.Printf("\n")
	}
	fmt.Println("---------------------------")
}

func listen_server(conn *net.Conn) {
	for {
		line := make([]byte, 4096)
		actual_size, err := (*conn).Read(line)
		if err != nil {
			str := fmt.Sprintf("reading error: %v", err)
			ch_logs <- str
			return
		}
		cmd := string(line[:actual_size])
		switch cmd {
		case "QUIT":
			return
		default:
			//fmt.Println("NEW TASK RECEIVE ...")
			task := &tsp_types.TaskType{}
			task.FromString(cmd)
			fmt.Println("Task size: ", task.Size, " matrix: ", len(*task.Matrix))
			print_matrix(task)
			//task.MinCost = tsp_types.POSITIVE_INF
			//task.CurrCost = tsp_types.DataType(0)
			answer_count++
			fmt.Printf("Calc answer (%d) ... ", answer_count)
			answer := tsp_solver.SolveImpl(*task)
			//fmt.Println("Answer: ", answer.Cost, " Jumps:", len(answer.Jumps))
			/*
				for i := 0; i < len(answer.Jumps); i++ {
					fmt.Printf(" %d-%d", answer.Jumps[i].Source, answer.Jumps[i].Destination)
				}
			*/
			//fmt.Println("Send: " + answer.ToString())
			fmt.Printf("%d ... ", answer.Cost)
			(*conn).Write([]byte(answer.ToString()))
			fmt.Printf("sent\n")
		}
	}
	defer (*conn).Close()
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
}

func main() {
	if len(os.Args) > 1 {
		worker_count, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Printf("Command Arg Error: %v\n", err)
		}
		fmt.Println("I'm try")
		var wg_logger sync.WaitGroup
		ch_logs = make(chan string, 100)
		ch_stop_logging := make(chan bool)
		wg_logger.Add(1)
		go log_thread(os.Stdout, ch_stop_logging, &wg_logger)
		for i := 0; i < worker_count; i++ {
			go start_worker()
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
				ch_stop_logging <- true
				wg_logger.Wait()
				fmt.Println("QUIT")
				return
			}
		}
	}
}
