package task_manager

import (
	//"bufio"
	"fmt"
	"net"
	"math/rand"
	"time"
	//"os"
	//"strconv"
	"container/list"
	tsp_solver "tsp_test/tsp/solver"
	tsp_types "tsp_test/tsp/types"
)

// --------------- GlobalCostType ---------------
type GlobalCostType struct {
	value tsp_types.DataType
	mutex chan bool
}

func (ai *GlobalCostType) Init(v tsp_types.DataType) {
	ai.value = v
	ai.mutex = make(chan bool, 1)
	ai.mutex <- true
}

func (ai *GlobalCostType) Get() tsp_types.DataType {
	<-ai.mutex
	v := ai.value
	ai.mutex <- true
	return v
}

func (ai *GlobalCostType) Set(v tsp_types.DataType) {
	<-ai.mutex
	ai.value = v
	ai.mutex <- true
}
//----------------------------------------------
// ----------- COUNTER --------------
type CounterType struct {
	value int
	mutex chan bool
}

func (ai *CounterType) Init(v int) {
	ai.value = v
	ai.mutex = make(chan bool, 1)
	ai.mutex <- true
}

func (ai *CounterType) Get() int {
	<-ai.mutex
	v := ai.value
	ai.mutex <- true
	return v
}

func (ai *CounterType) Inc() {
	<-ai.mutex
	ai.value++
	ai.mutex <- true
}
//-------------------------------------------
type ClientInfo struct {
	ID   int
	Conn *net.Conn
}

type WorkerInfo struct {
	ID          int
	Conn        *net.Conn
	CurrentTask *ClientTask
}

type ClientTask struct {
	Client ClientInfo
	Task   *tsp_types.TaskType
}

type TaskQueueItem struct {
	ID int
	Client ClientInfo
	SubTasks chan *tsp_types.TaskType
	SubAnswers chan *tsp_types.AnswerType
	FinalAnswer chan *tsp_types.AnswerType
	CurrMinCost GlobalCostType
	SubTaskCount int
	// for sync
	PreparedTasks chan bool
	PreparedAnswers chan bool
	SentTasksCount CounterType
	ReceivedAnswersCount CounterType
	PrepareFinished bool
	WorkerReady chan bool
}

func (tq_item *TaskQueueItem) Init(id int) {
	tq_item.ID = id
	tq_item.SubTasks = make(chan *tsp_types.TaskType, 20)
	tq_item.SubAnswers = make(chan *tsp_types.AnswerType, 20)
	tq_item.FinalAnswer = make(chan *tsp_types.AnswerType, 1)
	tq_item.CurrMinCost.Init(tsp_types.POSITIVE_INF)
	// for sync
	tq_item.PreparedTasks = make(chan bool, 20)
	tq_item.PreparedAnswers = make(chan bool, 20)
	tq_item.SentTasksCount.Init(0)
	tq_item.ReceivedAnswersCount.Init(0)
	tq_item.PrepareFinished = false
	tq_item.WorkerReady = make(chan bool, 1)
}

type TaskQueue struct {
	tasks *list.List
}

func (tq TaskQueue) Get(task_id int) {
	//
}
// ---------------------------------------------------------

var (
	CurrTask *TaskQueueItem
	free_workers_queue chan WorkerInfo
	ch_quit            chan bool
)

func CreateTaskManager() {
	//tasks_queue.Init()
	free_workers_queue = make(chan WorkerInfo, 100)
	ch_quit = make(chan bool, 1)
	CurrTask = &TaskQueueItem{}
	CurrTask.Init(0)
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

func AddFreeWorker(worker WorkerInfo) {
	CurrTask.WorkerReady <- true
	free_workers_queue <- worker
}

func AddNewTask(client ClientInfo, task_data []byte) {
	//
}

func print_matrix(matrix *tsp_types.MatrixType, size int) {
	fmt.Println("---------- MATRIX ---------")
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			fmt.Printf("%11d",int((*matrix)[i*size + j]))
		}
		fmt.Printf("\n")
	}
	fmt.Println("---------------------------")
}

func SolveTask(client ClientInfo, task_data []byte) {
	fmt.Println("SolveTask: Start")
	//task := generate_random_task_of_size_n(30, 100)
	task := tsp_types.TaskType{}
	task.FromXml(task_data)
	print_matrix(task.Matrix, task.Size)
	fmt.Println("SolveTask: Rand ", task.Size, len(*task.Matrix))
	
	go TaskHandler(CurrTask)
	go AnswerCombiner(CurrTask)

	//split
	enabled, task1, task2 := tsp_solver.CrusherImpl(&task)
	for enabled {
		print_matrix(task1.Matrix, task1.Size)
		fmt.Printf("CurrCost: %d, MinCost: %d, Jumps: %v\n", task1.CurrCost, task1.MinCost, task1.Jumps)
		CurrTask.SubTasks <- task1
		CurrTask.SentTasksCount.Inc()
		CurrTask.PreparedTasks <- true
		task2.MinCost = CurrTask.CurrMinCost.Get()
		enabled, task1, task2 = tsp_solver.CrusherImpl(task2)
	}
	CurrTask.PrepareFinished = true
	CurrTask.PreparedTasks <- false
	fmt.Println("[SolveTask] Split (",CurrTask.SentTasksCount.Get(),")")
	fmt.Println("[SolveTask] Finish")
}

func TaskHandler(tq_item *TaskQueueItem) {
	for i := 1; <-tq_item.PreparedTasks; i++ {
		worker := <-free_workers_queue
		<-tq_item.WorkerReady
		task := <-tq_item.SubTasks
		//tq_item.ReceivedAnswers <- true
		fmt.Printf("[TaskHandler] Sent %d task\n", i)
		// send to worker
		(*worker.Conn).Write([]byte(task.ToString()))
	}
	fmt.Println("[TaskHandler] Finish (count: ", tq_item.SentTasksCount.Get())
	//tq_item.ReceivedAnswers <- false
}

func AnswerHandler(answer_data []byte) {
	//not_empty := <-CurrTask.ReceivedAnswers
	CurrTask.ReceivedAnswersCount.Inc()
	fmt.Printf("[AnswerHandler] Receive %d answer ... ", CurrTask.ReceivedAnswersCount.Get())
	answer := tsp_types.AnswerType{}
	answer.FromString(string(answer_data))
	fmt.Printf("%d\n", answer.Cost)
	CurrTask.SubAnswers <- &answer
	CurrTask.PreparedAnswers <- true
	if (CurrTask.PrepareFinished && (CurrTask.ReceivedAnswersCount.Get() == CurrTask.SentTasksCount.Get())) {
		fmt.Println("[AnswerHandler] Finish")
		CurrTask.PreparedAnswers <- false
	}
}

func AnswerCombiner(tq_item *TaskQueueItem) {
	best_solution := &tsp_types.AnswerType{[]tsp_types.JumpType{}, tsp_types.POSITIVE_INF}
	for i := 1; <-tq_item.PreparedAnswers; i++ {
		possible_solution := <-tq_item.SubAnswers
		if (possible_solution.Cost != tsp_types.POSITIVE_INF) && (possible_solution.Cost < best_solution.Cost) {
			CurrTask.CurrMinCost.Set(possible_solution.Cost)
			best_solution = possible_solution
		}
		fmt.Printf("[AnswerCombiner] Combine %d answer\n", i)
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
