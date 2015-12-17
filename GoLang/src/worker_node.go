package main

import (
	//"bufio"
	"encoding/binary"
	"fmt"
	//"io"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	tsp_solver "tsp/solver"
	tsp_types "tsp/types"
)

type WorkerCmdType int

type CurrTaskInfo struct {
	TaskID  int
	MinCost tsp_types.GlobalCostType
}

var answer_count int = 0
var log_enable bool

func solve_task(task_data string, conn *net.Conn, logger *log.Logger, curr_task_info *CurrTaskInfo) {
	task := &tsp_types.TaskType{}
	task.FromString(task_data)
	answer_count++
	answer, _ := tsp_solver.SolveImpl(*task, &curr_task_info.MinCost)
	if log_enable {
		logger.Printf("Calc answer (%d) for task %d ... %d\n", answer_count, curr_task_info.TaskID, answer.Cost)
	}
	byte_answer := []byte(answer.ToString())
	err := binary.Write(*conn, binary.LittleEndian, int64(len(byte_answer)))
	if err != nil {
		logger.Printf("Write data error: %v", err)
		return
	}
	curr_task_info.TaskID = -1
	(*conn).Write([]byte(answer.ToString()))
}

func listen_server(conn *net.Conn, logger *log.Logger, curr_task_info *CurrTaskInfo) {
	for {
		var data_size int64
		err := binary.Read(*conn, binary.LittleEndian, &data_size)
		if err != nil {
			logger.Printf("Reading data size (client) error: %v\n", err)
			return
		}
		if log_enable {
			logger.Printf("Read new data\n")
		} else {
			fmt.Printf("\n")
		}
		data := make([]byte, data_size)
		_, err = (*conn).Read(data)
		switch string(data[0]) {
		case "q":
			return
		case "m":
			param_vec := strings.Split(string(data[1:]), " ")
			if len(param_vec) != 2 {
				logger.Println("min cost error: param count != 2")
				return
			}
			task_id, err := strconv.Atoi(param_vec[0])
			if err != nil {
				logger.Printf("task id convert error: %v\n", err)
				return
			}
			min_cost, err := strconv.Atoi(param_vec[1])
			if err != nil {
				logger.Printf("min cost convert error: %v\n", err)
				return
			}
			if curr_task_info.TaskID < 0 {
				curr_task_info.TaskID = task_id
				curr_task_info.MinCost.Set(tsp_types.DataType(min_cost))
			} else if task_id == curr_task_info.TaskID {
				curr_task_info.MinCost.Set(tsp_types.DataType(min_cost))
			}
		default:
			task_str := string(data[1:])
			sep_index := strings.Index(task_str, " ")
			new_task_id, err := strconv.Atoi(task_str[:sep_index])
			if err != nil {
				logger.Printf("curr task id convert error: %v\n", err)
				return
			}
			curr_task_info.TaskID = new_task_id
			go solve_task(task_str[sep_index+1:], conn, logger, curr_task_info)
		}
	}
	defer (*conn).Close()
}

func start_worker(worker_id int) {
	logger := log.New(os.Stdout, fmt.Sprintf("WORKER(%d): ", worker_id), log.Ldate|log.Ltime)
	gl_min_cost := tsp_types.GlobalCostType{}
	gl_min_cost.Init(tsp_types.POSITIVE_INF)
	curr_task_id := -1
	runtime.GOMAXPROCS(runtime.NumCPU())
	destination := "127.0.0.1:5000"
	logger.Println("worker: connect to ", destination)
	conn, err := net.Dial("tcp", destination)
	if err != nil {
		logger.Printf("dial error: %v", err)
	}
	curr_task_info := CurrTaskInfo{curr_task_id, gl_min_cost}
	go listen_server(&conn, logger, &curr_task_info)
}

func main() {
	log_enable = false
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "-l" {
			log_enable = true
			break
		}
	}
	if len(os.Args) > 1 {
		worker_count, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Printf("Command Arg Error: %v\n", err)
		}
		for i := 0; i < worker_count; i++ {
			go start_worker(i)
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
				fmt.Println("QUIT")
				return
			default:
				fmt.Println("New command: ", line)
			}
		}
	}
}
