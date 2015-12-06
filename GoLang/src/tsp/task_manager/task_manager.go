package task_manager

import (
	//"bufio"
	"fmt"
	"net"
	//"os"
	"strconv"
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
	Task *tsp_types.TaskType
	IsFinal bool
}

type WorkerFinalAnswer struct {
	Client ClientInfo
	Worker WorkerInfo
	Answer *tsp_types.AnswerType
}

type WorkerMediateAnswers struct {
	Worker WorkerInfo
	Answers chan *tsp_types.AnswerType
}

type AnswerTypeADT struct {
	id int
	channel chan *tsp_types.AnswerType
}

func (at AnswerTypeADT) eq(lhs *AnswerTypeADT, rhs *AnswerTypeADT) bool {
	return lhs.id == rhs.id
}

var (
	tasks_queue chan ClientTask
	ch_mediate_answers []WorkerMediateAnswers
	ch_final_answers chan WorkerFinalAnswer
	free_workers_queue chan WorkerInfo
	ch_quit chan bool
)

var output_chan_list *list.List
var output_chan_mutex chan int
var id_gen int = 10

func CreateTaskManager() {
	tasks_queue = make(chan ClientTask, 10)
	ch_mediate_answers = make([]WorkerMediateAnswers, 100)
	ch_final_answers = make(chan WorkerFinalAnswer, 10)
	free_workers_queue = make(chan WorkerInfo, 100)
	output_chan_list = list.New()
	output_chan_mutex = make(chan int, 1)
	output_chan_mutex <- 1
	ch_quit = make(chan bool, 1)
	for {
		select {
		case <-ch_quit:
			return
		default:
			ProcessingTask()
		}
	}
}

func AddNewWorker(worker WorkerInfo) {
	free_workers_queue <- worker
}

func SolveTask(client ClientInfo, task_data []byte, ch_logs chan string) {
	fmt.Println("SolveTask: Start " + string(task_data))
	task := tsp_types.TaskType{}
	task.FromXml(task_data)
	fmt.Println("SolveTask: Start 2",task.Size, len(task.Matrix))
	stack := make([]*tsp_types.TaskType, 1)
	fmt.Println("SolveTask: 121*")
	fmt.Println("SolveTask: 121")
	stack[0] = &task
	fmt.Println("SolveTask: 11")
	answer_adt := add_output_chan_list(ch_logs)
	fmt.Println("SolveTask: 12")
	//split
	for len(stack) > 0 {
		fmt.Println("Stack Len: " + strconv.Itoa(len(stack)))
		current_task := stack[len(stack)-1]
		fmt.Println("SolveTask: *1",current_task.Size, len(current_task.Matrix))
		if len(stack) == 1 {
			stack = make([]*tsp_types.TaskType, 0)
		} else {
			stack = stack[:len(stack)-1]
		}
		fmt.Println("SolveTask: *2")
		sub_task_info := tsp_solver.CrusherImpl(current_task)
		fmt.Println("SolveTask: *3")
		if sub_task_info.Enabled {
			stack = append(stack, sub_task_info.Task1)
			stack = append(stack, sub_task_info.Task2)
		} else {
			break
		}
		fmt.Println("SolveTask: *4")
	}
	fmt.Println("SolveTask: 13")
	// set to queue
	for i := 0; i < len(stack); i++ {
		tasks_queue <- ClientTask{client, stack[i], false}
	}
	//solve
	min_value := AtomicInt{}.new(int(tsp_types.POSITIVE_INF))
	num := make(chan bool, 10)
	number_of_tasks := len(stack)
	final_answer_channel := make(chan *tsp_types.AnswerType, 1)
	//*ch_logs <- "SolveTask: 1"
	go consumer(answer_adt.channel, number_of_tasks, final_answer_channel, *min_value, num)

	count := 0
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
	//collect
	//fmt.Println("[CLIENT] : COLLECT")
	best_solution := <-final_answer_channel
	//print answer
	if best_solution.Cost != tsp_types.POSITIVE_INF {
		//fmt.Println("***TASK SOLVED")
	} else {
		//fmt.Println("***TASK NOT SOLVE")
	}
	//*ch_logs <- "SolveTask: Finish"
}

func ProcessingTask() {
	worker := <- free_workers_queue
	/*
	// wait not "broken" worker
	for worker.Conn == nil {
		worker <- free_workers_queue
	}
	*/
	task := <- tasks_queue
	/*
	// wait not "broken" task
	for task.Client.Conn == nil {
		task <- tasks_queue
	}
	*/
	(*worker.Conn).Write([]byte(strconv.Itoa(task.Client.ID)+" "+strconv.Itoa(worker.ID)+" "+task.Task.ToString()+"\000"))
}

func consumer(ch_mediate_answer chan *tsp_types.AnswerType, number_of_tasks int, ch_final_answer chan *tsp_types.AnswerType, min_value AtomicInt, num chan bool) {
	best_solution := &tsp_types.AnswerType{[]tsp_types.JumpType{}, tsp_types.POSITIVE_INF}
	count := 0
	for i := 0; <-num; i++ {
		possible_solution := <-ch_mediate_answer
		count++
		if (possible_solution.Cost != tsp_types.POSITIVE_INF) && (possible_solution.Cost < best_solution.Cost) {
			min_value.set(int(possible_solution.Cost))
			best_solution = possible_solution
		}
	}
	ch_final_answer <- best_solution
}

func lock() {
	<-output_chan_mutex
}

func unlock() {
	output_chan_mutex <- 1
}

func add_output_chan_list(ch_logs chan string) *AnswerTypeADT {
	ch_logs <- "add_output_chan_list: Start"
	lock()
	ch_logs <- "add_output_chan_list: I'm do it"
	new_id := id_gen
	id_gen++
    channel := make(chan *tsp_types.AnswerType, 10)
	val := AnswerTypeADT{new_id, channel}
	output_chan_list.PushBack(&val)
	unlock()
	return &val
}

func delete_output_chan_list(val *AnswerTypeADT) {
	lock()
	for e := output_chan_list.Front(); e != nil; e = e.Next() {
		v := e.Value.(*AnswerTypeADT)
		if (*v).id == (*val).id {
			output_chan_list.Remove(e)
			break
		}
	}
	unlock()
}

/*
func ProcessingTask(task_data []byte) {
	task := tsp_types.TaskType{}
	task.FromXml(task_data)
	stack := make([]*tsp_types.TaskType, 0)
	tmp := task
	stack = append(stack, &tmp)

	answer := WorkerMediateAnswers{}
	number_of_tasks := 0
	//split
	for {
		current_task := stack[len(stack)-1]
		stack = stack[:len(stack)-2]
		sub_task_info := tsp_solver.CrusherImpl(*current_task)
		if sub_task_info.Enabled {
			stack = append(stack, &sub_task_info.Task1)
			stack = append(stack, &sub_task_info.Task2)
		} else {
			break
		}
	}

	//solve
	min_value := AtomicInt{}.new(int(tsp_types.POSITIVE_INF))
	num := make(chan int, 10)
	number_of_tasks = len(stack)
	final_answer_channel := make(chan *tsp_types.AnswerType, 1)
	go consumer(answer_adt.channel, number_of_tasks, final_answer_channel, *min_value, num)

	count := 0

	for stack != nil {
		current_task := stack[len(stack)-1]
		stack = stack[:len(stack)-2]
		min_val := min_value.get()
		fmt.Printf("[CLIENT] : sent new task id:%d count:%d cost:%d min_const: %d size: %d| of %d\n", count, answer_adt.id, current_task.SolutionCost, min_val, current_task.Size, number_of_tasks)
		count++
		if int(current_task.SolutionCost) > min_val {
			continue
		}
		new_task := current_task
		new_task.MinCost = tsp_types.DataType(min_val)
		task_queue <- new_task
		num <- true
	}
	num <- false
	//collect
	fmt.Println("[CLIENT] : COLLECT")
	best_solution := <-final_answer_channel
	//print answer
	if best_solution.Cost != tsp_types.POSITIVE_INF {
		fmt.Println("***TASK SOLVED")
	} else {
		fmt.Println("***TASK NOT SOLVE")
	}
}
*/