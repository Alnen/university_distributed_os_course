package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	tsp_solver "tsp/solver"
	tsp_types "tsp/types"
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
			return
		default:
			//fmt.Println("NEW TASK RECEIVE ...")
			task := tsp_types.TaskType{}
			task.FromString(cmd)
			//fmt.Println("Task size: ", task.Size, " matrix: ", len(task.Matrix))
			/*
				for i := 0; i < len(task.Matrix); i++ {
					fmt.Printf(" %d",int(task.Matrix[i]))
				}
				fmt.Println("")
			*/
			task.MinCost = tsp_types.POSITIVE_INF
			task.SolutionCost = 0
			answer := tsp_solver.SolveImpl(task)
			//fmt.Println("Answer: ", answer.Cost, " Jumps:", len(answer.Jumps))
			/*
				for i := 0; i < len(answer.Jumps); i++ {
					fmt.Printf(" %d-%d", answer.Jumps[i].Source, answer.Jumps[i].Destination)
				}
			*/
			//fmt.Println("Send: " + answer.ToString())
			(*conn).Write([]byte(answer.ToString() + "\000"))
			fmt.Println("Calc answer (", answer_count, "): ", answer.Cost)
			answer_count++
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
