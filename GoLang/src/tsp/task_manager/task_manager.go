package task_manager

import (
	//"bufio"
	"fmt"
	"math/rand"
	"net"
	"time"
	//"os"
	"container/list"
	"strconv"
	"encoding/binary"
	tsp_solver "tsp/solver"
	tsp_types "tsp/types"
)

type ClientInfo struct {
	ID   int
	Conn *net.Conn
}

type WorkerInfo struct {
	ID          int
	Conn        *net.Conn
	CurrentTask int
}

type ClientTask struct {
	Client ClientInfo
	Task   *tsp_types.TaskType
}

//-------------------- TaskQueueItem --------------------
type TaskQueueItem struct {
	ID           int
	Client       ClientInfo
	SubTasks     chan *tsp_types.TaskType
	SubAnswers   chan *tsp_types.AnswerType
	FinalAnswer  chan *tsp_types.AnswerType
	CurrMinCost  tsp_types.GlobalCostType
	SubTaskCount int
	// for sync
	PreparedTasks        chan bool
	PreparedAnswers      chan bool
	SentTasksCount       tsp_types.CounterType
	ReceivedAnswersCount tsp_types.CounterType
	PrepareFinished      bool
	WorkerReady          chan bool
}

func (tq_item *TaskQueueItem) Init(id int) {
	tq_item.ID = id
	tq_item.SubTasks = make(chan *tsp_types.TaskType, 10)
	tq_item.SubAnswers = make(chan *tsp_types.AnswerType, 10)
	tq_item.FinalAnswer = make(chan *tsp_types.AnswerType, 1)
	tq_item.CurrMinCost.Init(tsp_types.POSITIVE_INF)
	// for sync
	tq_item.PreparedTasks = make(chan bool, 10)
	tq_item.PreparedAnswers = make(chan bool, 10)
	tq_item.SentTasksCount.Init(0)
	tq_item.ReceivedAnswersCount.Init(0)
	tq_item.PrepareFinished = false
	tq_item.WorkerReady = make(chan bool, 1)
}

//-------------------- TaskQueue --------------------
type TaskQueue struct {
	tasks *list.List
	mutex chan bool
}

func (tq *TaskQueue) Init() {
	tq.tasks = list.New()
	tq.mutex = make(chan bool, 1)
	tq.mutex <- true
}

func (tq *TaskQueue) Get(task_id int) *TaskQueueItem {
	<-tq.mutex
	for el := tq.tasks.Front(); el != nil; el = el.Next() {
		task := el.Value.(*TaskQueueItem)
		if (*task).ID == task_id {
			tq.mutex <- true
			return task
		}
	}
	tq.mutex <- true
	return nil
}

func (tq *TaskQueue) Len() int {
	<-tq.mutex
	length := tq.tasks.Len()
	tq.mutex <- true
	return length
}

func (tq *TaskQueue) PushBack(new_item *TaskQueueItem) {
	<-tq.mutex
	tq.tasks.PushBack(new_item)
	tq.mutex <- true
}

func (tq *TaskQueue) PopFront() {
	<-tq.mutex
	tq.tasks.Remove(tq.tasks.Back())
	tq.mutex <- true
}

// ---------------------------------------------------------

var (
	task_queue TaskQueue
	//CurrTask           *TaskQueueItem
	free_workers_queue chan *WorkerInfo
	workers_list       *list.List
	clients_list       *list.List
	ch_quit            chan bool
)

func CreateTaskManager() {
	//tasks_queue.Init()
	free_workers_queue = make(chan *WorkerInfo, 100)
	ch_quit = make(chan bool, 1)
	workers_list = list.New()
	clients_list = list.New()
	task_queue.Init()
	//CurrTask = nil
	/*
		for {
			select {
			case <-ch_quit:
				return
			default:
				ProcessingTask()
			}
		}
	*/
}

func AddNewWorker(worker *WorkerInfo) {
	workers_list.PushBack(worker)
	free_workers_queue <- worker
}

func AddNewClient(client ClientInfo) {
	clients_list.PushBack(client)
}

func AddFreeWorker(worker *WorkerInfo) {
	free_workers_queue <- worker
}

func AddNewTask(client ClientInfo, task_data []byte) {
	//
}

func UpdateMinCost(task_id int, min_cost int) {
	for el := workers_list.Front(); el != nil; el = el.Next() {
		worker := el.Value.(*WorkerInfo)
		if task_id == (*worker).CurrentTask {
			msg := []byte(fmt.Sprintf("m%d %d", task_id, min_cost))
			err := binary.Write(*worker.Conn, binary.LittleEndian, int64(len(msg)))
			if err != nil {
			    fmt.Printf("[UpdateMinCost] Write data error: %v", err)
				return
			}
			(*worker.Conn).Write(msg)
		}
	}
}

func print_matrix(matrix *tsp_types.MatrixType, size int) {
	fmt.Println("---------- MATRIX ---------")
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			fmt.Printf("%11d", int((*matrix)[i*size+j]))
		}
		fmt.Printf("\n")
	}
	fmt.Println("---------------------------")
}

func SolveTask(client ClientInfo, task_data []byte, task_id int) {
	//CurrTask = &TaskQueueItem{}
	//CurrTask.Init(0)
	fmt.Println("SolveTask: Start")
	//task := generate_random_task_of_size_n(30, 100)
	task := tsp_types.TaskType{}
	task.FromXml(task_data)
	//print_matrix(task.Matrix, task.Size)
	fmt.Println("SolveTask: Rand ", task.Size, len(*task.Matrix))

	new_task := &TaskQueueItem{}
	new_task.Init(task_id)
	task_queue.PushBack(new_task)

	go TaskHandler(new_task)
	go AnswerCombiner(new_task)

	//split
	enabled, task1, task2 := tsp_solver.CrusherImpl(&task)
	for enabled {
		//print_matrix(task1.Matrix, task1.Size)
		//fmt.Printf("CurrCost: %d, MinCost: %d\n", task1.CurrCost, task1.MinCost)
		//fmt.Printf("CurrCost: %d, MinCost: %d, Jumps: %v\n", task1.CurrCost, task1.MinCost, task1.Jumps)
		new_task.SubTasks <- task1
		new_task.SentTasksCount.Inc()
		new_task.PreparedTasks <- true
		task2.MinCost = new_task.CurrMinCost.Get()
		enabled, task1, task2 = tsp_solver.CrusherImpl(task2)
	}
	new_task.PrepareFinished = true
	new_task.PreparedTasks <- false
	fmt.Println("[SolveTask] Split (", new_task.SentTasksCount.Get(), ")")
	final_answer := <-new_task.FinalAnswer
	fmt.Printf("[SolveTask] Final Answer: (cost: %d, jumps: %v\n", final_answer.Cost, final_answer.Jumps)
	
	msg := []byte(final_answer.ToXml())
	err := binary.Write(*client.Conn, binary.LittleEndian, int64(len(msg)))
	if err != nil {
	    fmt.Printf("[SolveTask] Write data error: %v", err)
		return
	}
	(*client.Conn).Write(msg)
	//fmt.Println("[SolveTask] Finish")
}

func TaskHandler(tq_item *TaskQueueItem) {
	for i := 1; <-tq_item.PreparedTasks; i++ {
		worker := <-free_workers_queue
		task := <-tq_item.SubTasks
		(*worker).CurrentTask = tq_item.ID
		msg := []byte("t" + strconv.Itoa(tq_item.ID) + " " + task.ToString())
		err := binary.Write(*worker.Conn, binary.LittleEndian, int64(len(msg)))
		if err != nil {
		    fmt.Printf("[TaskHandler] Write data error: %v", err)
			return
		}
		(*worker.Conn).Write(msg)
	}
	fmt.Println("[TaskHandler] Finish (count: ", tq_item.SentTasksCount.Get())
	//tq_item.ReceivedAnswers <- false
}

func AnswerHandler(task_id int, answer_data []byte) {
	//not_empty := <-CurrTask.ReceivedAnswers
	fmt.Printf("[AnswerHandler] Try Receive answer: task_id: %d, queue_len: %d\n", task_id, task_queue.Len())
	task := task_queue.Get(task_id)
	if task == nil {
		fmt.Println("[AnswerHandler] I'm not found ... ")
		return
	}
	task.ReceivedAnswersCount.Inc()
	fmt.Printf("[AnswerHandler] Receive %d answer ... ", task.ReceivedAnswersCount.Get())
	answer := tsp_types.AnswerType{}
	answer.FromString(string(answer_data))
	fmt.Printf("%d\n", answer.Cost)
	task.SubAnswers <- &answer
	task.PreparedAnswers <- true
	if task.PrepareFinished && (task.ReceivedAnswersCount.Get() == task.SentTasksCount.Get()) {
		fmt.Println("[AnswerHandler] Finish")
		task.PreparedAnswers <- false
	}
}

func AnswerCombiner(tq_item *TaskQueueItem) {
	best_solution := &tsp_types.AnswerType{[]tsp_types.JumpType{}, tsp_types.POSITIVE_INF}
	for i := 1; <-tq_item.PreparedAnswers; i++ {
		fmt.Printf("[AnswerCombiner] 1) Combine %d answer\n", i)
		possible_solution := <-tq_item.SubAnswers
		fmt.Printf("[AnswerCombiner] 2) Combine %d answer\n", i)
		if (possible_solution.Cost != tsp_types.POSITIVE_INF) && (possible_solution.Cost < best_solution.Cost) {
			tq_item.CurrMinCost.Set(possible_solution.Cost)
			UpdateMinCost(tq_item.ID, int(possible_solution.Cost))
			best_solution = possible_solution
		}
		//fmt.Printf("[AnswerCombiner] Combine %d answer\n", i)
	}
	fmt.Println("[AnswerCombiner] Finish")
	tq_item.FinalAnswer <- best_solution
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func generate_random_task_of_size_n(n int, modulus int) tsp_types.TaskType {
	rand.Seed(time.Now().UTC().UnixNano())
	var matrix tsp_types.MatrixType = make([]tsp_types.DataType, n*n)
	mapping := make([]int, n)
	for i := 0; i < n; i++ {
		mapping[i] = i
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i*n+j] = tsp_types.POSITIVE_INF
			} else {
				matrix[i*n+j] = tsp_types.DataType(randInt(1, modulus))
			}
		}
	}
	return tsp_types.TaskType{&matrix, mapping, mapping, []tsp_types.JumpType{}, tsp_types.DataType(0), tsp_types.POSITIVE_INF, n}
}
