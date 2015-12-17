package task_manager

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"net"
	//"strconv"
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
	fmt.Println("[TaskQueue.Get] task_id = ", task_id)
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
	tq.tasks.Remove(tq.tasks.Front())
	tq.mutex <- true
}

func (tq *TaskQueue) Remove(task_id int) {
	<-tq.mutex
	for el := tq.tasks.Front(); el != nil; el = el.Next() {
		task := el.Value.(*TaskQueueItem)
		if task.ID == task_id {
			tq.tasks.Remove(el)
			tq.mutex <- true
			return
		}
	}
	tq.mutex <- true
}

// ---------------------------------------------------------

var (
	task_queue         TaskQueue
	free_workers_queue chan *WorkerInfo
	workers_list       *list.List
	clients_list       *list.List
	ch_quit            chan bool
	LogEnable bool
)

func CreateTaskManager() {
	free_workers_queue = make(chan *WorkerInfo, 100)
	ch_quit = make(chan bool, 1)
	workers_list = list.New()
	clients_list = list.New()
	task_queue.Init()
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

func UpdateMinCost(task_id int, min_cost int) {
	for el := workers_list.Front(); el != nil; el = el.Next() {
		worker := el.Value.(*WorkerInfo)
		if task_id == (*worker).CurrentTask {
			msg := []byte(fmt.Sprintf("m%d %d", task_id, min_cost))
			err := binary.Write(*worker.Conn, binary.LittleEndian, int64(len(msg)))
			if err != nil {
				fmt.Printf("[UpdateMinCost] Write data error: %v\n", err)
				return
			}
			(*worker.Conn).Write(msg)
		}
	}
}

func UpdateOneWorkerMinCost(worker *WorkerInfo, task_id int, min_cost int) {
	for el := workers_list.Front(); el != nil; el = el.Next() {
		w := el.Value.(*WorkerInfo)
		if worker.ID == w.ID {
			msg := []byte(fmt.Sprintf("m%d %d", task_id, min_cost))
			err := binary.Write(*worker.Conn, binary.LittleEndian, int64(len(msg)))
			if err != nil {
				fmt.Printf("[UpdateOneWorkerMinCost] Write data error: %v\n", err)
				return
			}
			(*worker.Conn).Write(msg)
			return
		}
	}
}

func SolveTask(client ClientInfo, task_data []byte, task_id int) {
	fmt.Println("SolveTask: Start")
	task := tsp_types.TaskType{}
	task.FromXml(task_data)

	new_task := &TaskQueueItem{}
	new_task.Init(task_id)
	task_queue.PushBack(new_task)

	go TaskHandler(new_task)
	go AnswerCombiner(new_task)

	//split
	enabled, task1, task2 := tsp_solver.CrusherImpl(&task)
	for enabled {
		new_task.SubTasks <- task1
		new_task.SentTasksCount.Inc()
		new_task.PreparedTasks <- true
		task2.MinCost = new_task.CurrMinCost.Get()
		enabled, task1, task2 = tsp_solver.CrusherImpl(task2)
	}
	new_task.PrepareFinished = true
	new_task.PreparedTasks <- false
	final_answer := <-new_task.FinalAnswer

	msg := []byte(final_answer.ToXml())
	err := binary.Write(*client.Conn, binary.LittleEndian, int64(len(msg)))
	if err != nil {
		fmt.Printf("[SolveTask] Write data error: %v", err)
		return
	}
	(*client.Conn).Write(msg)
	task_queue.Remove(new_task.ID)
	fmt.Println("[SolveTask] Finish")
}

func TaskHandler(tq_item *TaskQueueItem) {
	for i := 1; <-tq_item.PreparedTasks; i++ {
		worker := <-free_workers_queue
		task := <-tq_item.SubTasks
		(*worker).CurrentTask = tq_item.ID
		UpdateOneWorkerMinCost(worker, tq_item.ID, int(tsp_types.POSITIVE_INF))
		msg := []byte(fmt.Sprintf("t%d %s", tq_item.ID, task.ToString()))
		//msg := []byte("t" + strconv.Itoa(tq_item.ID) + " " + task.ToString())
		err := binary.Write(*worker.Conn, binary.LittleEndian, int64(len(msg)))
		if err != nil {
			fmt.Printf("[TaskHandler] Write data error: %v", err)
			return
		}
		(*worker.Conn).Write(msg)
	}
	if LogEnable {
		fmt.Println("[TaskHandler] Finish (count: ", tq_item.SentTasksCount.Get())
	}
}

func AnswerHandler(task_id int, answer_data []byte) {
	task := task_queue.Get(task_id)
	if task == nil {
		fmt.Println("[AnswerHandler] I'm not found ... ")
		return
	}
	task.ReceivedAnswersCount.Inc()
	answer := tsp_types.AnswerType{}
	answer.FromString(string(answer_data))
	if LogEnable {
		fmt.Printf("[AnswerHandler] Receive %d answer ... %d\n", task.ReceivedAnswersCount.Get(), answer.Cost)
	}
	task.SubAnswers <- &answer
	task.PreparedAnswers <- true
	if task.PrepareFinished && (task.ReceivedAnswersCount.Get() == task.SentTasksCount.Get()) {
		if LogEnable {
			fmt.Println("[AnswerHandler] Finish")
		}
		task.PreparedAnswers <- false
	}
}

func AnswerCombiner(tq_item *TaskQueueItem) {
	best_solution := &tsp_types.AnswerType{[]tsp_types.JumpType{}, tsp_types.POSITIVE_INF}
	for i := 1; <-tq_item.PreparedAnswers; i++ {
		possible_solution := <-tq_item.SubAnswers
		if (possible_solution.Cost != tsp_types.POSITIVE_INF) && (possible_solution.Cost < best_solution.Cost) {
			tq_item.CurrMinCost.Set(possible_solution.Cost)
			UpdateMinCost(tq_item.ID, int(possible_solution.Cost))
			best_solution = possible_solution
		}
		if LogEnable {
			fmt.Printf("[AnswerCombiner] Combine %d answer\n", i)
		}
	}
	if LogEnable {
		fmt.Println("[AnswerCombiner] Finish")
	}
	tq_item.FinalAnswer <- best_solution
}
