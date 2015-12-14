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

func calculate_plus_cost(matrix tsp_types.MatrixType, n int) (bool, tsp_types.DataType) {
	cost := tsp_types.DataType(0)
	// every row
	for i := 0; i < n; i++ {
		min_value := tsp_types.POSITIVE_INF
		// looking min value in row
		infinity_count := 0
		for j := 0; j < n; j++ {
			if matrix[i*n+j] == tsp_types.POSITIVE_INF {
				infinity_count++
			} else if matrix[i*n+j] < min_value {
				min_value = matrix[i*n+j]
			}
		}
		// if all elements in row are infinite then return
		if infinity_count == n {
			return false, tsp_types.DataType(0)
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
		min_value := tsp_types.POSITIVE_INF
		// looking min value in collum
		infinity_count := 0
		for j := 0; j < n; j++ {
			if matrix[j*n+i] == tsp_types.POSITIVE_INF {
				infinity_count++
			} else if matrix[j*n+i] < min_value {
				min_value = matrix[j*n+i]
			}
		}
		// if all elements in collum are infinite then return
		if infinity_count == n {
			return false, tsp_types.DataType(0)
		}
		if min_value != 0 {
			// subtract min value from entire column
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

func find_heavier_zero(matrix tsp_types.MatrixType, n int) tsp_types.ZeroInfoType {
	heavier_zero := tsp_types.ZeroInfoType{0, 0, tsp_types.NEGATIVE_INF}
	for i := 0; i < n; i++ {
		// for every zero in row
		for j := 0; j < n; j++ {
			if matrix[i*n+j] == 0 {
				// find min element in row
				weight := tsp_types.DataType(0)
				min_value := tsp_types.POSITIVE_INF
				for z := 0; z < n; z++ {
					if (z != j) && (matrix[i*n+z] < min_value) && (matrix[i*n+z] != tsp_types.POSITIVE_INF) {
						min_value = matrix[i*n+z]
					}
				}
				if min_value != tsp_types.POSITIVE_INF {
					weight += min_value
				}
				// find min element in collum
				min_value = tsp_types.POSITIVE_INF
				for z := 0; z < n; z++ {
					if (z != i) && (matrix[z*n+j] < min_value) && (matrix[z*n+j] != tsp_types.POSITIVE_INF) {
						min_value = matrix[z*n+j]
					}
				}
				if min_value != tsp_types.POSITIVE_INF {
					weight += min_value
				}
				// remember new zero
				if heavier_zero.Weight < weight {
					heavier_zero = tsp_types.ZeroInfoType{i, j, weight}
				}
			}
		}
	}
	return heavier_zero
}

func find_previous_jump(all_jumps []tsp_types.JumpType, current_jump tsp_types.JumpType) (tsp_types.JumpType, bool) {
	for i := 0; i < len(all_jumps); i++ {
		if all_jumps[i].Destination == current_jump.Source {
			return all_jumps[i], true
		}
	}
	return current_jump, false
}

func find_next_jump(all_jumps []tsp_types.JumpType, current_jump tsp_types.JumpType) (tsp_types.JumpType, bool) {
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
		//fmt.Printf("[find_last_jump_in_direction] curr: %v jumps: %v\n", current_jump, all_jumps)
		switch direction {
		case tsp_types.FORWARD_DIR:
			last_jump, found_new = find_next_jump(all_jumps, last_jump)
		case tsp_types.BACKWARD_DIR:
			last_jump, found_new = find_previous_jump(all_jumps, last_jump)
		default:
			fmt.Println("i dont know what is happening")
			return tsp_types.JumpType{-1, -1}, false
		}
		if found_new == false {
			break
		}
	}
	if (current_jump.Source != last_jump.Source) && (current_jump.Destination != last_jump.Destination) {
		return last_jump, true
	} else {
		return last_jump, false
	}
}

func forbid_jump_if_needed(matrix *tsp_types.MatrixType, x_mapping []int, y_mapping []int, all_jumps []tsp_types.JumpType, n int) {
	var inf_x, inf_y int
	beginning_of_chain, found_next := find_last_jump_in_direction(all_jumps[1:], all_jumps[0], tsp_types.BACKWARD_DIR)
	//fmt.Printf("[forbid_jump_if_needed]* \n")
	end_of_chain, found_prev := find_last_jump_in_direction(all_jumps[1:], all_jumps[0], tsp_types.FORWARD_DIR)
	//fmt.Printf("[forbid_jump_if_needed]** \n")
	if !(found_prev && found_next) {
		for i := 0; i < n; i++ {
			if x_mapping[i] == end_of_chain.Destination {
				inf_x = i
			}
			if y_mapping[i] == beginning_of_chain.Source {
				inf_y = i
			}
		}
		(*matrix)[inf_x*n+inf_y] = tsp_types.POSITIVE_INF
	}
}

func generate_sub_task_data(matrix *tsp_types.MatrixType, x_mapping []int, y_mapping []int, all_jumps []tsp_types.JumpType, zero_with_most_weight tsp_types.ZeroInfoType, n int) tsp_types.SubTaskDataType {
	var new_matrix tsp_types.MatrixType = make([]tsp_types.DataType, (n-1)*(n-1))
	new_x_mapping := make([]int, n-1)
	new_y_mapping := make([]int, n-1)
	new_all_jumps := make([]tsp_types.JumpType, 1)

	zero_x := zero_with_most_weight.Row
	zero_y := zero_with_most_weight.Col
	new_jump := tsp_types.JumpType{
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
				new_matrix[i_x*(n-1)+j_y] = (*matrix)[i*n+j]
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
	//fmt.Printf("[generate]** \n")
	forbid_jump_if_needed(&new_matrix, new_x_mapping, new_y_mapping, new_all_jumps, n-1)
	return tsp_types.SubTaskDataType{&new_matrix, new_x_mapping, new_y_mapping, new_all_jumps}
}

//var SolveImpl_counter int = 0

func SolveImpl(task tsp_types.TaskType, gl_min_cost *tsp_types.GlobalCostType) (tsp_types.AnswerType, bool) {
	//fmt.Println("[**] Task size: ", task.Size," matrix size: ",len(*task.Matrix), " cost:", int(task.SolutionCost), " min:", int(task.MinCost))
	//SolveImpl_counter++
	//fmt.Printf("[SolveImpl] %d\n", SolveImpl_counter)
	//fmt.Printf("[SolveImpl] Task y mapping: %v\n", task.YMapping)
	//fmt.Printf("[SolveImpl] Task jumps: %v\n", task.Jumps)
	branch_count++
	if gl_min_cost.Get() < task.MinCost {
		return tsp_types.ERROR_ANSWER, false
	}
	if (task.Size == 0) || (task.Size == 1) {
		//fmt.Println("----- ERROR ANSWER 1")
		return tsp_types.ERROR_ANSWER, true
	}
	success, additional_cost := calculate_plus_cost(*task.Matrix, task.Size)
	//fmt.Printf("additional_cost: %d\n", additional_cost)
	if !success {
		//fmt.Println("----- ERROR ANSWER 2")
		return tsp_types.ERROR_ANSWER, true
	}
	task.CurrCost += additional_cost
	heavier_zero := find_heavier_zero(*task.Matrix, task.Size)
	//fmt.Printf("[SolveImpl] heavier_zero: %v\n", heavier_zero)
	// if task.Size is 2 end recursion
	if task.Size == 2 {
		next_city := heavier_zero.Col
		previous_city := heavier_zero.Row
		if ((*task.Matrix)[(previous_city^1)*2+(next_city^1)] == 0) &&
			((*task.Matrix)[(previous_city^1)*2+next_city] == tsp_types.POSITIVE_INF) &&
			((*task.Matrix)[previous_city*2+(next_city^1)] == tsp_types.POSITIVE_INF) {
			//sys->print("END COST : %d", cost);
			jumps := make([]tsp_types.JumpType, 2)
			jumps[0] = tsp_types.JumpType{task.RowMapping[heavier_zero.Row], task.ColMapping[heavier_zero.Col]}
			jumps[1] = tsp_types.JumpType{task.RowMapping[heavier_zero.Row^1], task.ColMapping[heavier_zero.Col^1]}
			for j := range task.Jumps {
				jumps = append(jumps, task.Jumps[j])
			}
			//fmt.Println("SOLUTION COST: ", int(task.SolutionCost))
			return tsp_types.AnswerType{jumps, task.CurrCost}, true
		} else {
			//fmt.Println("tsp_types.ERROR_ANSWER 1")
			//fmt.Println("----- ERROR ANSWER 3")
			return tsp_types.ERROR_ANSWER, true
		}
	}
	//fmt.Printf("[SolveImpl]* \n")
	// prepare data for recursive call
	sub_dt := generate_sub_task_data(task.Matrix, task.RowMapping, task.ColMapping, task.Jumps, heavier_zero, task.Size)
	//call this function recursively
	//fmt.Printf("[SolveImpl]** \n")
	answer, less_global := SolveImpl(tsp_types.TaskType{sub_dt.Matrix, sub_dt.RowMapping, sub_dt.ColMapping, sub_dt.Jumps, task.CurrCost, task.MinCost, task.Size - 1}, gl_min_cost)
	if !less_global {
		return tsp_types.ERROR_ANSWER, false
	}
	//print_answer(answer, "GOT ANSWER")
	final_path := []tsp_types.JumpType{}
	if answer.Cost < task.MinCost {
		//fmt.Printf("[SolveImpl] AnswerJumps: %v\n", answer.Jumps)
		final_path = answer.Jumps
		task.MinCost = answer.Cost
	}
	//right_path
	if (task.CurrCost + heavier_zero.Weight) < task.MinCost {
		//correct first one
		(*task.Matrix)[heavier_zero.Row*task.Size+heavier_zero.Col] = tsp_types.POSITIVE_INF
		answer, less_global = SolveImpl(task, gl_min_cost)
		if !less_global {
			return tsp_types.ERROR_ANSWER, false
		}
		if answer.Cost < task.MinCost {
			final_path = answer.Jumps
			task.MinCost = answer.Cost
		}
	}
	// return answer
	if task.MinCost < tsp_types.POSITIVE_INF {
		//fmt.Printf("[SolveImpl] %i | %v\n", int(task.SolutionCost), final_path)
		return tsp_types.AnswerType{final_path, task.MinCost}, true
	} else {
		//fmt.Println("[SolveImpl] ERROR_ANSWER")
		//fmt.Println("----- ERROR ANSWER 4")
		return tsp_types.ERROR_ANSWER, true
	}
}

//var crusher_counter int = 0

func CrusherImpl(task *tsp_types.TaskType) (bool, *tsp_types.TaskType, *tsp_types.TaskType) {
	//crusher_counter++
	//fmt.Printf("[CrusherImpl] (%d) size : %d, matrix size %d\n", crusher_counter, task.Size, len(task.Matrix))
	success, additional_cost := calculate_plus_cost(*task.Matrix, task.Size)
	if success != true {
		return false, nil, nil
	}
	task.CurrCost += additional_cost
	heavier_zero := find_heavier_zero(*task.Matrix, task.Size)
	//prepare data for recursive call
	sub_dt := generate_sub_task_data(task.Matrix, task.RowMapping, task.ColMapping, task.Jumps, heavier_zero, task.Size)
	//call this function recursively
	task1 := tsp_types.TaskType{sub_dt.Matrix, sub_dt.RowMapping, sub_dt.ColMapping, sub_dt.Jumps, task.CurrCost, task.MinCost, task.Size - 1}
	(*task.Matrix)[heavier_zero.Row*task.Size+heavier_zero.Col] = tsp_types.POSITIVE_INF
	return true, &task1, task
}
