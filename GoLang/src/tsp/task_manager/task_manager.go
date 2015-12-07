package task_manager

import (
	//"bufio"
	"fmt"
	"net"
	//"os"
	//"strconv"
	"container/list"
	tsp_solver "tsp/solver"
	tsp_types "tsp/types"
)

type AtomicInt struct {
	value int
	mutex chan int
}

func (ai AtomicInt) new(val int) *AtomicInt {
	ai.value = val
	ai.mutex = make(chan int, 1)
	ai.mutex <- 1
	return &ai
}

func (ai AtomicInt) get() int {
	<-ai.mutex
	v := ai.value
	ai.mutex <- 1
	return v
}

func (ai AtomicInt) set(val int) {
	<-ai.mutex
	ai.value = val
	ai.mutex <- 1
	return
}

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

type TaskQueue struct {
	tasks       *list.List
	CurrentTask *TaskQueueElem
	mutex       chan bool
}

func (tq *TaskQueue) Init() {
	tq.CurrentTask = nil
	tq.tasks = list.New()
	tq.mutex = make(chan bool, 1)
	tq.mutex <- true
}

func (tq *TaskQueue) Lock() {
	<-tq.mutex
}

func (tq *TaskQueue) Unlock() {
	tq.mutex <- true
}

func (tq *TaskQueue) PushBack(tq_elem *TaskQueueElem) {
	tq.Lock()
	if tq.CurrentTask == nil {
		tq.CurrentTask = tq_elem
	}
	tq.tasks.PushBack(tq_elem)
	tq.Unlock()
}

func (tq *TaskQueue) PopFront() *TaskQueueElem {
	tq.Lock()
	if tq.tasks.Front() == nil {
		tq.Unlock()
		return nil
	}
	remove_el := tq.tasks.Front().Value.(*TaskQueueElem)
	tq.tasks.Remove(tq.tasks.Front())
	v := tq.tasks.Front()
	if v != nil {
		tq.CurrentTask = v.Value.(*TaskQueueElem)
	} else {
		tq.CurrentTask = nil
	}
	tq.Unlock()
	return remove_el
}

type TaskQueueElem struct {
	Task        ClientTask
	SubTasks    chan *tsp_types.TaskType
	SubAnswers  chan *tsp_types.AnswerType
	FinalAnswer chan *tsp_types.AnswerType
	Counter     chan bool
}

func (tq_elem *TaskQueueElem) Init(task ClientTask) {
	tq_elem.Task = task
	tq_elem.SubTasks = make(chan *tsp_types.TaskType, 20)
	tq_elem.SubAnswers = make(chan *tsp_types.AnswerType, 20)
	tq_elem.FinalAnswer = make(chan *tsp_types.AnswerType, 1)
	tq_elem.Counter = make(chan bool, 20)
}

var (
	tasks_queue        TaskQueue
	free_workers_queue chan WorkerInfo
	ch_quit            chan bool
	solved_subtasks    int
)

var output_chan_list *list.List
var output_chan_mutex chan int
var id_gen int = 10

func CreateTaskManager() {
	tasks_queue.Init()
	free_workers_queue = make(chan WorkerInfo, 100)
	ch_quit = make(chan bool, 1)
	solved_subtasks = 0
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
	free_workers_queue <- worker
}

func consumer(task_queue_element *TaskQueueElem, number_of_tasks int, min_value AtomicInt) {
	best_solution := &tsp_types.AnswerType{[]tsp_types.JumpType{}, tsp_types.POSITIVE_INF}
	count := 0
	for i := 0; true; i++ {
		x := <-task_queue_element.Counter
		//fmt.Println("[consumer] Counter (", count, "): ", x)
		if x == false {
			break
		}
		possible_solution := <-task_queue_element.SubAnswers
		//fmt.Println("[consumer] get possible_solution (", count, ")")
		count++
		//fmt.Println("[consumer] Get answer: (", count, ")", possible_solution.Cost)
		if (possible_solution.Cost != tsp_types.POSITIVE_INF) && (possible_solution.Cost < best_solution.Cost) {
			min_value.set(int(possible_solution.Cost))
			best_solution = possible_solution
		}
		//fmt.Println("Counter is work 3")
	}
	fmt.Println("Wait best solution...")
	task_queue_element.FinalAnswer <- best_solution
}

func SolveTask(client ClientInfo, task_data []byte, ch_logs chan string) {
	fmt.Println("SolveTask: Start " + string(task_data))
	task := tsp_types.TaskType{}
	task.FromXml(task_data)
	task_queue_element := TaskQueueElem{}
	task_queue_element.Init(ClientTask{client, &task})
	fmt.Println("SolveTask: Start 2", task.Size, len(task.Matrix))
	stack := make([]*tsp_types.TaskType, 1)
	fmt.Println("SolveTask: 121*")
	fmt.Println("SolveTask: 121")
	stack[0] = &task
	fmt.Println("SolveTask: 11")
	//answer_adt := add_output_chan_list(ch_logs)
	fmt.Println("SolveTask: 12")
	//split
	for len(stack) > 0 {
		//fmt.Println("Stack Len: " + strconv.Itoa(len(stack)))
		current_task := stack[len(stack)-1]
		//fmt.Println("SolveTask: *1",current_task.Size, len(current_task.Matrix))
		if len(stack) == 1 {
			stack = make([]*tsp_types.TaskType, 0)
		} else {
			stack = stack[:len(stack)-1]
		}
		//fmt.Println("SolveTask: *2")
		sub_task_info := tsp_solver.CrusherImpl(current_task)
		//fmt.Println("SolveTask: *3")
		if sub_task_info.Enabled {
			stack = append(stack, sub_task_info.Task1)
			stack = append(stack, sub_task_info.Task2)
		} else {
			break
		}
		//fmt.Println("SolveTask: *4")
	}
	fmt.Println("SolveTask: 13 ", len(stack))
	//solve
	min_value := AtomicInt{}.new(int(tsp_types.POSITIVE_INF))
	//num := make(chan bool, 20)
	number_of_tasks := len(stack)
	//final_answer_channel := make(chan *tsp_types.AnswerType, 1)
	fmt.Println("SolveTask: 14")
	//*ch_logs <- "SolveTask: 1"
	tasks_queue.PushBack(&task_queue_element)

	go ProcessingTask(number_of_tasks)
	go consumer(&task_queue_element, number_of_tasks, *min_value)

	// set to queue
	fmt.Println("[SolveTask] Task Count: ", number_of_tasks, " Stack size:", len(stack))
	for i := 0; i < number_of_tasks; i++ {
		task_queue_element.SubTasks <- stack[i]
		task_queue_element.Counter <- true
	}
	fmt.Println(".......... FININSH COUNTER ............")
	task_queue_element.Counter <- false
	/*
		count := 0
		fmt.Println("SolveTask: 15")
		//*ch_logs <- "SolveTask: 2"
		for stack != nil {
			current_task := stack[len(stack)-1]
			stack = stack[:len(stack)-2]
			min_val := min_value.get()
			//fmt.Printf("[CLIENT] : sent new task id:%d count:%d cost:%d min_const: %d size: %d| of %d\n", count, answer_adt.id, current_task.SolutionCost, min_val, current_task.Size, number_of_tasks)
			count++
			if int(current_task.SolutionCost) > min_val {
				continue
			}
			new_task := current_task
			new_task.MinCost = tsp_types.DataType(min_val)
			tasks_queue <- ClientTask{client, new_task, false}
			num <- true
		}
		num <- false
	*/
	//collect
	fmt.Println("[CLIENT] : COLLECT")
	best_solution := <-task_queue_element.FinalAnswer
	// write answer to client
	//print answer
	if best_solution.Cost != tsp_types.POSITIVE_INF {
		fmt.Println("***TASK SOLVED")
	} else {
		fmt.Println("***TASK NOT SOLVE")
	}
	fmt.Println("Finish")
	//*ch_logs <- "SolveTask: Finish"
}

func ProcessingTask(task_count int) {
	fmt.Println("[ProcessingTask] Task count: ", task_count)
	if tasks_queue.CurrentTask != nil {
		for i := 0; i < task_count; i++ {
			//fmt.Println("I'm find new task (", i, ")")
			worker := <-free_workers_queue
			//fmt.Println("I'm find new worker(", i, "): ", worker.ID)
			task := <-tasks_queue.CurrentTask.SubTasks
			//fmt.Println("I'm find new subtask task")
			(*worker.Conn).Write([]byte(task.ToString() + "\000"))
		}
		fmt.Println("[ProcessingTask] FINISH ", task_count)
	}
}

func ProcessingAnswer(answer_data []byte) {
	answer := tsp_types.AnswerType{}
	answer.FromString(string(answer_data))
	//fmt.Println("[ProcessingAnswer] Answer received(", solved_subtasks, "): ", answer.Cost)
	tasks_queue.CurrentTask.SubAnswers <- &answer
	fmt.Print("[ProcessingAnswer] Answer (", solved_subtasks, ") sent to queue")
	solved_subtasks++
}
