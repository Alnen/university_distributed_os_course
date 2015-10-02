implement dts_solver;

include "sys.m";
	sys: Sys;
include "draw.m";
include "rand.m";
	rand: Rand;
include "math.m";
	math: Math;

# custom types
DataType : type int;
IndexType : type int;
MatrixType : type array of DataType;
ZeroInfoType : type (IndexType, IndexType, DataType);
JumpType : type (IndexType, IndexType);
SubTaskDataType : type (MatrixType, array of IndexType, array of IndexType, list of JumpType);
AnswerType : type (list of JumpType, DataType);
DirectionType : type int;
BoolType : type int;
DataTaskType : type (MatrixType, array of IndexType, array of IndexType, IndexType);
# custom constants
FORWARD_DIR : con 1;
BACKWARD_DIR : con -1;
POSITIVE_INF : con 2147483647;
NEGATIVE_INF : con -2147483647;
NVAL_INDEX : con -1;
ERROR_ANSWER : con (nil, POSITIVE_INF);
False : con 0;
True  : con 1;

#interface
dts_solver: module {
	init: fn(nil: ref Draw->Context, argv: list of string);
	solve(matrix: MatrixType) : AnswerType;
};


#implementation
init(nil: ref Draw->Context, argv: list of string)
{
	sys = load Sys Sys->PATH;
	rand = load Rand Rand->PATH;
	math = load Math Math->PATH;

	if (len argv == 2)
	{
		case hd tl argv
		{
			"test" =>
				run_test_case();
			"benchmark" => 
				run_benchmark();
			* =>
				raise "Unknown argument";
		}
	}
	else
	{
		raise "Expected arguments : \"test\" or \"benchmark\"";
	}
	
}

print_square_matrix(matrix : MatrixType, n : IndexType)
{
	sys->print("=========MATRIX=========\n");
	for (i := IndexType 0; i < n; ++i)
	{
		for (j := IndexType 0; j < n; ++j)
		{
			if (matrix[i*n + j] == POSITIVE_INF)
			{
				sys->print("%4s ", "INF");
			}
			else
			{
				sys->print("%4d ", matrix[i*n + j]);
			}
		}
		sys->print("\n");
	}
	sys->print("========================\n");
}

print_array(container : array of IndexType, name : string)
{
	sys->print("=========%s=========\n", name);
	for (i := IndexType 0; i < len container; ++i)
	{
		sys->print("%d ", container[i]);
	}
	sys->print("\n");
}

debug_print(matrix : MatrixType, x_mapping : array of IndexType,  y_mapping  : array of IndexType, zero_with_most_weight : ZeroInfoType, n : IndexType)
{
	sys->print("ZERO : i(%d:%d) j(%d:%d) weight(%d)\n", 
		zero_with_most_weight.t0, 
		x_mapping[zero_with_most_weight.t0] + 1,
		zero_with_most_weight.t1, 
		y_mapping[zero_with_most_weight.t1] + 1, 
		zero_with_most_weight.t2
	);
	print_array(x_mapping, "X_MAPPING");
	print_array(y_mapping, "Y_MAPPING");
	print_square_matrix(matrix, n);
	sys->print("==================================================================================\n");
}

print_answer(answer : AnswerType, title : string)
{
	begin : IndexType;
	end : IndexType;

	tail := answer.t0;
	sys->print("=========%s=========\n", title);
	while(tail != nil)
	{
		head := hd tail;
		tail = tl tail;

		(begin, end) = head;
        sys->print("%d - %d \n",begin ,end);
	}
	sys->print("cost:  %d \n",answer.t1);
}

calculate_additional_cost_and_correct_matrix(matrix: MatrixType, n: IndexType) : (BoolType, DataType)
{
	cost : DataType = 0;
	# every row
	for (i := IndexType 0; i < n; ++i)
	{
		min_value : DataType = POSITIVE_INF;
		# looking min value in row
		infinity_count : IndexType = 0;
		for (j := IndexType 0; j < n; ++j)
		{
			if (matrix[i*n + j] == POSITIVE_INF)
			{
				++infinity_count;
			}
			else if (matrix[i*n + j] < min_value)
			{
				min_value = matrix[i*n + j];
			}
		}
		# if all elements in row are infinite then return
		if (infinity_count == n)
		{
			return (False, 0);
		}
		if (min_value != 0){
			# subtract min value from entire row
			for (j := IndexType 0; j < n; ++j)
			{
				if (matrix[i*n + j] != POSITIVE_INF) matrix[i*n + j] -= min_value;
			}
			# add min value to cost
			cost += min_value;
		}
	}
	# every collum
	for (i = IndexType 0; i < n; ++i)
	{
		min_value : DataType = POSITIVE_INF;
		# looking min value in collum
		infinity_count : IndexType = 0;
		for (j := IndexType 0; j < n; ++j)
		{
			if (matrix[j*n + i] == POSITIVE_INF)
			{
				++infinity_count;
			}
			else if (matrix[j*n + i] < min_value)
			{
				min_value = matrix[j*n + i];
			}
		}
		# if all elements in collum are infinite then return
		if (infinity_count == n)
		{
			return (False, 0);
		}
		if (min_value != 0){
			# subtract min value from entire collum
			for (j := IndexType 0; j < n; ++j)
			{
				if (matrix[j*n + i] != POSITIVE_INF) matrix[j*n + i] -= min_value;
			}
			# add min value to cost
			cost += min_value;
		}
	}
	return (True, cost);
}

find_zero_with_biggest_weight(matrix : MatrixType, n : IndexType) : ZeroInfoType
{
	zero_with_most_weight : ZeroInfoType = (0, 0, NEGATIVE_INF);
	for (i := IndexType 0; i < n; ++i)
	{
		# for every zero in row
		for (j := IndexType 0; j < n; ++j)
		{
			if (matrix[i*n + j] == 0)
			{
				# find min element in row
				weight : DataType = 0;
				min_value : DataType = POSITIVE_INF;
				for (z := IndexType 0; z < n; ++z){
					if (z != j && matrix[i*n + z] < min_value && matrix[i*n + z] != POSITIVE_INF) min_value = matrix[i*n + z];
				}
				if (min_value != POSITIVE_INF) weight += min_value;
				# find min element in collum
				min_value = POSITIVE_INF;
				for (z = IndexType 0; z < n; ++z){
					if (z != i && matrix[z*n + j] < min_value && matrix[z*n + j] != POSITIVE_INF) min_value = matrix[z*n + j];
				}
				if (min_value != POSITIVE_INF) weight += min_value;
				# remember new zero
				#sys->print("POSSIBLE ZERO : i(%d) j(%d) weight(%d)\n", i, j, weight);
				if (zero_with_most_weight.t2 < weight) {
					#sys->print(" CHANGED");
					zero_with_most_weight = (i, j, weight);
				}
				#sys->print("\n");
			}
		}
	}
	##sys->print("FINAL ZERO : i(%d) j(%d) weight(%d)\n", zero_with_most_weight.t0, zero_with_most_weight.t1, zero_with_most_weight.t2);
	return zero_with_most_weight;
}

find_previous_jump_in_chain(all_jumps : list of JumpType, current_jump : JumpType) : (JumpType, BoolType)
{
	found := False;
	tail := all_jumps;

	while(tail != nil)
	{
		head := hd tail;
		tail = tl tail;
		if (head.t1 == current_jump.t0)
		{
			return (head, True);
		}
	}
	return (current_jump, False);
}

find_next_jump_in_chain(all_jumps : list of JumpType, current_jump : JumpType) : (JumpType, BoolType)
{
	found := False;
	tail := all_jumps;

	while(tail != nil)
	{
		head := hd tail;
		tail = tl tail;
		if (head.t0 == current_jump.t1)
		{
			return (head, True);
		}
	}
	return (current_jump, False);
}

find_last_jump_in_direction(all_jumps : list of JumpType, current_jump : JumpType, direction : DirectionType) : (JumpType, BoolType)
{
	found_new := False;
	last_jump := current_jump;

	do
	{
		case direction
		{
			FORWARD_DIR =>
				(last_jump, found_new) = find_next_jump_in_chain(all_jumps, last_jump);
			BACKWARD_DIR =>
				(last_jump, found_new) = find_previous_jump_in_chain(all_jumps, last_jump);
			* =>
				raise "i dont know what is happening";
		}
	}
	while(found_new);

	if(current_jump.t0 != last_jump.t0 && current_jump.t1 != last_jump.t1)
	{
		return (last_jump, True);
	}
	else
	{
		return (last_jump, False);
	}
}

forbid_jump_if_needed(matrix : MatrixType, x_mapping : array of IndexType, y_mapping: array of IndexType, all_jumps : list of JumpType, n : IndexType)
{ 
	#sys->print("++++++++++++++++++\n");
	#print_square_matrix(matrix, n);
	#sys->print("++++++++++++++++++\n");
	
	inf_x : IndexType;
	inf_y : IndexType;

	#sys->print("++++++++++++++++++\n");
	#print_container(x_mapping, "X_MAPPING");
	#print_container(y_mapping, "Y_MAPPING");
	#print_answer((jumps, -1), "FIX_DEBUG");
	#sys->print("JUMP: (%d,%d)\n", last_jump.t0, last_jump.t1);
	#sys->print("N: %d\n", n);
	
	(beginning_of_chain, found_next) := find_last_jump_in_direction(tl all_jumps, hd all_jumps, BACKWARD_DIR);
	(end_of_chain, found_prev) := find_last_jump_in_direction(tl all_jumps, hd all_jumps, FORWARD_DIR);

	#sys->print("FOUND: %d and %d \n",found_prev, found_next);
	if (!(found_prev && found_next))
	{
		for (i := 0; i < n; ++i)
		{
			if(x_mapping[i] == end_of_chain.t1) inf_x = i;
			if(y_mapping[i] == beginning_of_chain.t0) inf_y = i;
		}

		#sys->print("FORBID: %d(%d) -> %d(%d) \n",end_of_chain.t1, inf_x,  beginning_of_chain.t0, inf_y);
		matrix[inf_x*n + inf_y] = POSITIVE_INF;
	}
	#print_square_matrix(matrix, n);
	#sys->print("++++++++++++++++++\n");
}

generate_sub_task_data(matrix : MatrixType, x_mapping : array of IndexType, y_mapping : array of IndexType, all_jumps : list of JumpType, zero_with_most_weight : ZeroInfoType, n : IndexType) : SubTaskDataType
{
	new_matrix := array[(n - 1)*(n - 1)] of DataType;
	new_x_mapping := array[n - 1] of IndexType;
	new_y_mapping := array[n - 1] of IndexType;
	new_all_jumps : list of JumpType;

	zero_x : IndexType = zero_with_most_weight.t0;
	zero_y : IndexType = zero_with_most_weight.t1;
	searching_x : IndexType = y_mapping[zero_y];
	searching_y : IndexType = x_mapping[zero_x];

	new_jump : JumpType = (
			x_mapping[zero_with_most_weight.t0],
			y_mapping[zero_with_most_weight.t1]
	);
	new_all_jumps = new_jump :: all_jumps;

	# prepare new x_mapping, y_mapping and matrix.
	# if you think it's looks ungly it's because it is.
	i_x := IndexType 0;
	i_y := IndexType 0;
	for (i := IndexType 0; i < n; ++i){
		if (i != zero_x){
			new_x_mapping[i_x] = x_mapping[i];
			j_y := IndexType 0;
			for (j := IndexType 0; j < n; ++j) {
				if (j == zero_y){
					continue;
				}
				new_matrix[i_x*(n - 1) + j_y] = matrix[i*n + j];
				++j_y;
			}
		}else{
			--i_x;
		}
		if (i != zero_y){
			new_y_mapping[i_y] = y_mapping[i];
		}else{
			--i_y;
		}
		++i_x;
		++i_y;
	}

	forbid_jump_if_needed(new_matrix, new_x_mapping, new_y_mapping, new_all_jumps, n-1);
	return (new_matrix, new_x_mapping, new_y_mapping, new_all_jumps);
}

solve_impl(matrix : MatrixType, x_mapping : array of IndexType, y_mapping : array of IndexType, all_jumps: list of JumpType, solution_cost : DataType, min_cost : DataType, n : IndexType): AnswerType
{
	#sys->print("CURR N: %d COST : %d\n", n, solution_cost);
	if (n == 0 || n == 1)
	{
		return ERROR_ANSWER;
	}

	(success, additional_cost) := calculate_additional_cost_and_correct_matrix(matrix, n);
	if (!success)
	{
		#debug_print(matrix, x_mapping, y_mapping, (0,0,0), n);
		#sys->print("TROLOLO\n");
		return ERROR_ANSWER;
	}
	solution_cost += additional_cost;
	zero_with_most_weight : ZeroInfoType = find_zero_with_biggest_weight(matrix, n);
	##sys->print("FINAL OUT ZERO : i(%d) j(%d) weight(%d)\n", zero_with_most_weight.t0, zero_with_most_weight.t1, zero_with_most_weight.t2);
	#debug_print(matrix, x_mapping, y_mapping, zero_with_most_weight, n);

	# if n is 2 end recursion
	if (n == 2)
	{
		next_city : IndexType = zero_with_most_weight.t1;
		previous_city : IndexType = zero_with_most_weight.t0;

		if (matrix[(previous_city ^ 1) * 2 + (next_city ^ 1)] == 0 &&
			matrix[(previous_city ^ 1) * 2 + next_city] == POSITIVE_INF &&
			matrix[previous_city * 2 + (next_city ^ 1)] == POSITIVE_INF)
		{
			#sys->print("END COST : %d", cost);
			return (
				(
					x_mapping[zero_with_most_weight.t0],
					y_mapping[zero_with_most_weight.t1]
				) 
				::
				(
					x_mapping[zero_with_most_weight.t0 ^ 1],
					y_mapping[zero_with_most_weight.t1 ^ 1]
				) 
				:: 
				all_jumps
				,
				solution_cost
			);
		}
		else
		{
			return ERROR_ANSWER;
		}
	}
	# prepare data for recursive call
	(new_matrix, new_x_mapping, new_y_mapping, new_all_jumps) := generate_sub_task_data(matrix, x_mapping, y_mapping, all_jumps, zero_with_most_weight, n);

	#call this function recursively
	answer : AnswerType = solve_impl(new_matrix, new_x_mapping, new_y_mapping, new_all_jumps, solution_cost, min_cost, n - 1);

	#print_answer(answer, "GOT ANSWER");
	final_path : list of JumpType = nil;
	if (answer.t1 < min_cost)
	{
		(final_path, min_cost) = answer;
		#print_answer((end_of_path, min_cost), "NEW ANSWER");
	}
	#right_path
	if (solution_cost + zero_with_most_weight.t2 < min_cost)
	{
		#correct first one
		matrix[zero_with_most_weight.t0*n + zero_with_most_weight.t1] = POSITIVE_INF;

		answer = solve_impl(matrix, x_mapping, y_mapping, all_jumps, solution_cost, min_cost, n);
		#print_answer(answer, "GOT ANSWER");
		if (answer.t1 < min_cost)
		{
			(final_path, min_cost) = answer;
		}
		#print_answer((end_of_path, min_cost), "NEW ANSWER");
	}
	# return answer
	if (min_cost < POSITIVE_INF)
	{
		return (final_path, min_cost);
	}
	else
	{
		return ERROR_ANSWER;
	}
}

generate_random_task_of_size_n(n: IndexType, modulus: int, seed: int) : DataTaskType
{
	rand->init(seed);

	matrix := array[n*n] of DataType;
	mapping := array[n] of IndexType;

	for (i := IndexType 0; i < n ; ++i)
	{
		mapping[i] = i;
		for(j := 0; j < n; ++j)
		{
			if (i == j)
			{
				matrix[i*n + j] = POSITIVE_INF;
			}
			else
			{
				matrix[i*n + j] = rand->rand(modulus);
			}
		}
	}

	return (matrix, mapping, mapping, n);
}

test_case_task() : DataTaskType
{
	n : IndexType = 5;

	matrix := array[] of
	{
		POSITIVE_INF,	25,					40,					31,					27,
		5,					POSITIVE_INF,	17,					30,					25,
		19,					15,					POSITIVE_INF,	6,					1,
		9,					50,					24,					POSITIVE_INF,	6,
		22,					8,					7,					10,					POSITIVE_INF
	};

	mapping := array[n] of IndexType;
	for (i := IndexType 0; i < n; ++i){
		mapping[i] = i;
	}

	return (matrix, mapping, mapping, n);
}

solve(matrix: MatrixType) : AnswerType
{
	n := int math->sqrt(real len matrix);
	mapping := array[n] of IndexType;
	for (i := IndexType 0; i < n; ++i){
		mapping[i] = i;
	}
	return solve_impl(matrix, mapping, mapping, nil, 0, POSITIVE_INF, n);
}

run_test_case()
{
	t1, t2 : int;
	task := test_case_task();
	print_square_matrix(task.t0, 5);
	t1 = sys->millisec();
	answer : AnswerType = solve_impl(task.t0, task.t1, task.t2, nil, 0, POSITIVE_INF, task.t3);
	t2 = sys->millisec();
	print_answer(answer, "FINAL-ANSWER");
	sys->print("It took %d msec\n ", t2-t1);
}

run_benchmark()
{
	t1, t2 : int;
	for(i := 5; i < 50 ; ++i)
	{
		task := generate_random_task_of_size_n(i, 100, 0);
		t1 = sys->millisec();
		answer : AnswerType = solve_impl(task.t0, task.t1, task.t2, nil, 0, POSITIVE_INF, task.t3);
		t2 = sys->millisec();
		sys->print("For i : %d it took %d msec\n", i, t2-t1);
	}
	
}