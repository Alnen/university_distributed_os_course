package solver

import (
	"fmt"
	"os"
	"strconv"
	tsp_types "tsp/types"
)

var branch_count int = 0

func print_square_matrix(matrix tsp_types.MatrixType, n int) {
	f, err := os.Create("./res01.txt")
	if err != nil {
		panic(err)
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if matrix[i*n+j] == tsp_types.POSITIVE_INF {
				str := fmt.Sprintf("%4s", "INF")
				f.WriteString(str)
				//fmt.Printf("%4s ", "INF")
			} else {
				str := strconv.Itoa(int(matrix[i*n+j]))
				str = fmt.Sprintf("%4s", str)
				f.WriteString(str)
				//fmt.Printf("%4d ", matrix[i*int(n) + j])
			}
		}
		f.WriteString("\n")
		//fmt.Println("")
	}
}

func print_array(container []int, name string) {
	fmt.Printf("=========%s=========\n", name)
	for i := 0; i < len(container); i++ {
		fmt.Printf("%d ", container[i])
	}
	fmt.Println("")
}

func debug_print(matrix tsp_types.MatrixType, x_mapping []int, y_mapping []int, zero_with_most_weight tsp_types.ZeroInfoType, n int) {
	fmt.Printf("ZERO : i(%d:%d) j(%d:%d) weight(%d)\n",
		zero_with_most_weight.Row,
		x_mapping[zero_with_most_weight.Row]+1,
		zero_with_most_weight.Col,
		y_mapping[zero_with_most_weight.Col]+1,
		zero_with_most_weight.Weight,
	)
	print_array(x_mapping, "X_MAPPING")
	print_array(y_mapping, "Y_MAPPING")
	print_square_matrix(matrix, n)
	fmt.Println("==================================================================================")
}

func print_answer(answer tsp_types.AnswerType, title string) {
	fmt.Printf("=========%s=========\n", title)
	for j := range answer.Jumps {
		fmt.Println("%d - %d \n", answer.Jumps[j].Source, answer.Jumps[j].Destination)
	}
	fmt.Printf("cost:  %d \n", answer.Cost)
}

func calculate_additional_cost_and_correct_matrix(matrix tsp_types.MatrixType, n int) (r1 bool, r2 tsp_types.DataType) {
	var cost tsp_types.DataType = 0
	// every row
	for i := 0; i < n; i++ {
		var min_value tsp_types.DataType = tsp_types.POSITIVE_INF
		// looking min value in row
		var infinity_count int = 0
		for j := 0; j < n; j++ {
			if matrix[i*n+j] == tsp_types.POSITIVE_INF {
				infinity_count++
			} else if matrix[i*n+j] < min_value {
				min_value = matrix[i*n+j]
			}
		}
		// if all elements in row are infinite then return
		if infinity_count == n {
			return false, 0
		}
		if min_value != 0 {
			// subtract min value from entire row
			for j := 0; j < n; j++ {
				if matrix[i*n+j] != tsp_types.POSITIVE_INF {
					matrix[i*n+j] -= min_value
				}
			}
			// add min value to cost
			cost += min_value
		}
	}
	// every collum
	for i := 0; i < n; i++ {
		var min_value tsp_types.DataType = tsp_types.POSITIVE_INF
		// looking min value in collum
		var infinity_count int = 0
		for j := 0; j < n; j++ {
			if matrix[j*n+i] == tsp_types.POSITIVE_INF {
				infinity_count++
			} else if matrix[j*n+i] < min_value {
				min_value = matrix[j*n+i]
			}
		}
		// if all elements in collum are infinite then return
		if infinity_count == n {
			return false, 0
		}
		if min_value != 0 {
			// subtract min value from entire collum
			for j := 0; j < n; j++ {
				if matrix[j*n+i] != tsp_types.POSITIVE_INF {
					matrix[j*n+i] -= min_value
				}
			}
			// add min value to cost
			cost += min_value
		}
	}
	return true, cost
}

func find_zero_with_biggest_weight(matrix tsp_types.MatrixType, n int) tsp_types.ZeroInfoType {
	var zero_with_most_weight tsp_types.ZeroInfoType = tsp_types.ZeroInfoType{0, 0, tsp_types.NEGATIVE_INF}
	for i := 0; i < n; i++ {
		// for every zero in row
		for j := 0; j < n; j++ {
			if matrix[i*n+j] == 0 {
				// find min element in row
				var weight tsp_types.DataType = 0
				var min_value tsp_types.DataType = tsp_types.POSITIVE_INF
				for z := 0; z < n; z++ {
					if z != j && matrix[i*n+z] < min_value && matrix[i*n+z] != tsp_types.POSITIVE_INF {
						min_value = matrix[i*n+z]
					}
				}
				if min_value != tsp_types.POSITIVE_INF {
					weight += min_value
				}
				// find min element in collum
				min_value = tsp_types.POSITIVE_INF
				for z := 0; z < n; z++ {
					if z != i && matrix[z*n+j] < min_value && matrix[z*n+j] != tsp_types.POSITIVE_INF {
						min_value = matrix[z*n+j]
					}
				}
				if min_value != tsp_types.POSITIVE_INF {
					weight += min_value
				}
				// remember new zero
				//fmt.Printf("POSSIBLE ZERO : i(%d) j(%d) weight(%d)\n", i, j, weight)
				if zero_with_most_weight.Weight < weight {
					//fmt.Printf(" CHANGED")
					zero_with_most_weight = tsp_types.ZeroInfoType{i, j, weight}
				}
				//fmt.Println("")
			}
		}
	}
	//fmt.Printf("FINAL ZERO : i(%d) j(%d) weight(%d)\n", zero_with_most_weight.t0, zero_with_most_weight.t1, zero_with_most_weight.t2)
	return zero_with_most_weight
}

func find_previous_jump_in_chain(all_jumps []tsp_types.JumpType, current_jump tsp_types.JumpType) (r1 tsp_types.JumpType, r2 bool) {
	for i := 0; i < len(all_jumps); i++ {
		if all_jumps[i].Destination == current_jump.Source {
			return all_jumps[i], true
		}
	}
	return current_jump, false
}

func find_next_jump_in_chain(all_jumps []tsp_types.JumpType, current_jump tsp_types.JumpType) (r1 tsp_types.JumpType, r2 bool) {
	for i := 0; i < len(all_jumps); i++ {
		if all_jumps[i].Source == current_jump.Destination {
			return all_jumps[i], true
		}
	}
	return current_jump, false
}

func find_last_jump_in_direction(all_jumps []tsp_types.JumpType, current_jump tsp_types.JumpType, direction tsp_types.DirectionType) (r1 tsp_types.JumpType, r2 bool) {
	found_new := false
	last_jump := current_jump
	for {
		switch direction {
		case tsp_types.FORWARD_DIR:
			last_jump, found_new = find_next_jump_in_chain(all_jumps, last_jump)
		case tsp_types.BACKWARD_DIR:
			last_jump, found_new = find_previous_jump_in_chain(all_jumps, last_jump)
		default:
			fmt.Println("i dont know what is happening")
			return
		}
		if found_new == false {
			break
		}
	}
	if current_jump.Source != last_jump.Source && current_jump.Destination != last_jump.Destination {
		return last_jump, true
	} else {
		return last_jump, false
	}
}

func forbid_jump_if_needed(matrix tsp_types.MatrixType, x_mapping []int, y_mapping []int, all_jumps []tsp_types.JumpType, n int) {
	var inf_x int
	var inf_y int
	beginning_of_chain, found_next := find_last_jump_in_direction(all_jumps[1:], all_jumps[0], tsp_types.BACKWARD_DIR)
	end_of_chain, found_prev := find_last_jump_in_direction(all_jumps[1:], all_jumps[0], tsp_types.FORWARD_DIR)
	if !(found_prev && found_next) {
		for i := 0; i < n; i++ {
			if x_mapping[i] == end_of_chain.Destination {
				inf_x = i
			}
			if y_mapping[i] == beginning_of_chain.Source {
				inf_y = i
			}
		}
		matrix[inf_x*n+inf_y] = tsp_types.POSITIVE_INF
	}
}

func generate_sub_task_data(matrix tsp_types.MatrixType, x_mapping []int, y_mapping []int, all_jumps []tsp_types.JumpType, zero_with_most_weight tsp_types.ZeroInfoType, n int) tsp_types.SubTaskDataType {
	new_matrix := make([]tsp_types.DataType, (n-1)*(n-1))
	new_x_mapping := make([]int, n-1)
	new_y_mapping := make([]int, n-1)
	new_all_jumps := make([]tsp_types.JumpType, 1)

	zero_x := zero_with_most_weight.Row
	zero_y := zero_with_most_weight.Col
	var new_jump tsp_types.JumpType = tsp_types.JumpType{
		x_mapping[zero_with_most_weight.Row],
		y_mapping[zero_with_most_weight.Col],
	}
	new_all_jumps[0] = new_jump
	for j := range all_jumps {
		new_all_jumps = append(new_all_jumps, all_jumps[j])
	}
	// prepare new x_mapping, y_mapping and matrix.
	i_x := 0
	i_y := 0
	for i := 0; i < n; i++ {
		if i != zero_x {
			new_x_mapping[i_x] = x_mapping[i]
			j_y := 0
			for j := 0; j < n; j++ {
				if j == zero_y {
					continue
				}
				new_matrix[i_x*(n-1)+j_y] = matrix[i*n+j]
				j_y++
			}
		} else {
			i_x--
		}
		if i != zero_y {
			new_y_mapping[i_y] = y_mapping[i]
		} else {
			i_y--
		}
		i_x++
		i_y++
	}
	forbid_jump_if_needed(new_matrix, new_x_mapping, new_y_mapping, new_all_jumps, n-1)
	return tsp_types.SubTaskDataType{new_matrix, new_x_mapping, new_y_mapping, new_all_jumps}
}

func SolveImpl(task tsp_types.TaskType) tsp_types.AnswerType {
//func SolveImpl(matrix tsp_types.MatrixType, x_mapping []int, y_mapping []int, all_jumps []tsp_types.JumpType, solution_cost tsp_types.DataType, min_cost tsp_types.DataType, n int) tsp_types.AnswerType {
	branch_count++
	if task.Size == 0 || task.Size == 1 {
		return tsp_types.ERROR_ANSWER
	}
	success, additional_cost := calculate_additional_cost_and_correct_matrix(task.Matrix, task.Size)
	if !success {
		return tsp_types.ERROR_ANSWER
	}
	task.SolutionCost += additional_cost
	zero_with_most_weight := find_zero_with_biggest_weight(task.Matrix, task.Size)
	// if task.Size is 2 end recursion
	var answer tsp_types.AnswerType
	if task.Size == 2 {
		next_city := zero_with_most_weight.Col
		previous_city := zero_with_most_weight.Row
		if (task.Matrix[(previous_city^1)*2+(next_city^1)] == 0) &&
			(task.Matrix[(previous_city^1)*2+next_city] == tsp_types.POSITIVE_INF) &&
			(task.Matrix[previous_city*2+(next_city^1)] == tsp_types.POSITIVE_INF) {
			//sys->print("END COST : %d", cost);
			var jumps = make([]tsp_types.JumpType, 2)
			jumps[0] = tsp_types.JumpType{task.XMapping[zero_with_most_weight.Row], task.YMapping[zero_with_most_weight.Col]}
			jumps[1] = tsp_types.JumpType{task.XMapping[zero_with_most_weight.Row^1], task.YMapping[zero_with_most_weight.Col^1]}
			for j := range task.Jumps {
				jumps = append(jumps, task.Jumps[j])
			}
			return tsp_types.AnswerType{jumps, task.SolutionCost}
		} else {
			return tsp_types.ERROR_ANSWER
		}
	}
	// prepare data for recursive call
	sub_dt := generate_sub_task_data(task.Matrix, task.XMapping, task.YMapping, task.Jumps, zero_with_most_weight, task.Size)
	//call this function recursively
	answer = SolveImpl(tsp_types.TaskType{sub_dt.Matrix, sub_dt.XMapping, sub_dt.YMapping, sub_dt.Jumps, task.SolutionCost, task.MinCost, task.Size-1})
	//print_answer(answer, "GOT ANSWER")
	final_path := []tsp_types.JumpType{}
	if answer.Cost < task.MinCost {
		final_path = answer.Jumps
		task.MinCost = answer.Cost
	}
	//right_path
	if task.SolutionCost+zero_with_most_weight.Weight < task.MinCost {
		//correct first one
		task.Matrix[zero_with_most_weight.Row*task.Size+zero_with_most_weight.Col] = tsp_types.POSITIVE_INF
		answer = SolveImpl(task)
		if answer.Cost < task.MinCost {
			final_path = answer.Jumps
			task.MinCost = answer.Cost
		}
	}
	// return answer
	if task.MinCost < tsp_types.POSITIVE_INF {
		return tsp_types.AnswerType{final_path, task.MinCost}
	} else {
		return tsp_types.ERROR_ANSWER
	}
}

func CrusherImpl(task tsp_types.TaskType) tsp_types.DivTaskType {
	success, additional_cost := calculate_additional_cost_and_correct_matrix(task.Matrix, task.Size)
	if success != true {
		return tsp_types.DivTaskType{false, tsp_types.ERROR_TASK, tsp_types.ERROR_TASK}
	}
	task.SolutionCost += additional_cost
	var zero_with_most_weight tsp_types.ZeroInfoType = find_zero_with_biggest_weight(task.Matrix, task.Size)
	//prepare data for recursive call
	sub_dt := generate_sub_task_data(task.Matrix, task.XMapping, task.YMapping, task.Jumps, zero_with_most_weight, task.Size)
	//call this function recursively
	task1 := tsp_types.TaskType{sub_dt.Matrix, sub_dt.XMapping, sub_dt.YMapping, sub_dt.Jumps, task.SolutionCost, task.MinCost, task.Size - 1}
	task.Matrix[zero_with_most_weight.Row*task.Size+zero_with_most_weight.Col] = tsp_types.POSITIVE_INF
	return tsp_types.DivTaskType{true, task1, task}
}
