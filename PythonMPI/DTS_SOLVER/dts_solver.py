import numpy as np
import collections
import itertools
import enum
import time

__author__ = 'Alex Uzhegov'

ZeroInfoType = collections.namedtuple('ZeroInfoType', ['row', 'column', 'weight'])
JumpType = collections.namedtuple('JumpType', ['begin', 'end'])
TaskType = collections.namedtuple('TaskType',
                                  [
                                      'matrix',
                                      'row_mapping',
                                      'column_mapping',
                                      'jumps',
                                      'current_weight',
                                      'minimum_weight',
                                      'size'
                                  ])
DataTaskType = collections.namedtuple('DataTaskType',
                                      [
                                          'matrix',
                                          'row_mapping',
                                          'column_mapping'
                                      ])
SubTaskDataType = collections.namedtuple('SubTaskDataType',
                                         [
                                             'matrix',
                                             'row_mapping',
                                             'column_mapping',
                                             'jumps'
                                         ])
AnswerType = collections.namedtuple('AnswerType', ['jumps', 'cost'])


class Direction(enum.Enum):
    FORWARD = 1
    BACKWARD = -1


# custom constants
POSITIVE_INF = np.iinfo(np.int32).max
NEGATIVE_INF = np.iinfo(np.int32).min
NVAL_INDEX = -1
ERROR_ANSWER = AnswerType([], POSITIVE_INF)


# Func
# solve: fn(matrix: MatrixType) : AnswerType;
#    generate_random_task_of_size_n: fn(n: int, modulus: int, seed: int) : DataTaskType;
#    crusher_impl: fn(matrix : MatrixType, x_mapping : array of int, y_mapping : array of int, all_jumps: list of JumpType, solution_cost : int, min_cost : int, n : int): (int, (TaskType, TaskType));
#    solve_impl: fn(matrix : MatrixType, x_mapping : array of int, y_mapping : array of int, all_jumps: list of JumpType, solution_cost : int, min_cost : int, n : int): AnswerType;


def debug_print(matrix, x_mapping, y_mapping, zero_with_most_weight):
    print('=============MATRIX=================')
    print(matrix)
    print('=============MAPPING_X==============')
    print(x_mapping)
    print('=============MAPPING_Y==============')
    print(y_mapping)
    print('ZERO : i({0}:{1}) j({2}:{3}) weight({4})'.format(
        zero_with_most_weight.row,
        x_mapping[zero_with_most_weight.row] + 1,
        zero_with_most_weight.column,
        y_mapping[zero_with_most_weight.column] + 1,
        zero_with_most_weight.weight
    ))
    print("=========END===================================================================\n")


def print_answer(answer: AnswerType):
    print('=========ANSWER=========')
    for begin, end in answer.jumps:
        print('{} - {}'.format(begin, end))
    print("cost: ", answer.cost)
    print('========================')

#@profile
def calculate_additional_cost_and_correct_matrix(matrix):
    row_number, column_number = matrix.shape
    additional_cost = 0

    for i in range(row_number):
        infinity_count = 0
        min_value = POSITIVE_INF

        for j in range(column_number):
            if matrix.item(i, j) == POSITIVE_INF:
                infinity_count += 1
            elif matrix.item(i, j) < min_value:
                min_value = matrix.item(i, j)

        if infinity_count == row_number:
            return False, 0

        if min_value != 0:
            for j in range(column_number):
                if matrix.item(i, j) != POSITIVE_INF:
                    matrix[i][j] -= min_value
            additional_cost += min_value

    for i in range(column_number):
        infinity_count = 0
        min_value = POSITIVE_INF

        for j in range(row_number):
            if matrix.item(j, i) == POSITIVE_INF:
                infinity_count += 1
            elif matrix.item(j, i) < min_value:
                min_value = matrix.item(j, i)

        if infinity_count == row_number:
            return False, 0

        if min_value != 0:
            for j in range(column_number):
                if matrix.item(j, i) != POSITIVE_INF:
                    matrix[j][i] -= min_value
            additional_cost += min_value

    return True, additional_cost

#@profile
def find_zero_with_biggest_weight(matrix):
    row_size, column_size = matrix.shape
    zero_with_most_weight = ZeroInfoType(0, 0, NEGATIVE_INF)
    zero_index = np.where(matrix == 0)
    for (i, j) in zip(zero_index[0], zero_index[1]):
        weight = 0

        row = matrix[i]
        row_with_filtered_infinities = row[np.where(row != POSITIVE_INF)]
        filtered_current_element = row_with_filtered_infinities[np.where(row_with_filtered_infinities != j)]
        if len(filtered_current_element) != 0:
            min_value = filtered_current_element.min()
            if min_value != POSITIVE_INF:
                weight += min_value

        min_value = POSITIVE_INF
        for z in range(matrix.shape[1]):
            value = matrix.item(z, j)
            if z != i and value < min_value and value != POSITIVE_INF:
                min_value = value
        if min_value != POSITIVE_INF:
            weight += min_value
        """
        column = matrix[:, j]
        column_with_filtered_infinities = column[np.where(column != POSITIVE_INF)]
        column_with_filtered_current_element = column_with_filtered_infinities[np.where(column_with_filtered_infinities != i)]
        if len(column_with_filtered_current_element) != 0:
            min_value = column_with_filtered_current_element.min()
            if min_value != POSITIVE_INF:
                weight += min_value
        """

        if zero_with_most_weight.weight < weight:
            zero_with_most_weight = ZeroInfoType(i, j, weight)
    return zero_with_most_weight


def find_previous_jump_in_chain(all_jumps, current_jump):
    for begin, end in all_jumps:
        if end == current_jump.begin:
            return True, JumpType(begin, end)
    return False, current_jump


def find_next_jump_in_chain(all_jumps, current_jump):
    for begin, end in all_jumps:
        if begin == current_jump.end:
            return True, JumpType(begin, end)
    return False, current_jump


def find_last_jump_in_direction(all_jumps, current_jump, direction):
    find_next_jump_func_mapping = {
        Direction.FORWARD: find_next_jump_in_chain,
        Direction.BACKWARD: find_previous_jump_in_chain
    }

    last_jump = current_jump
    while True:
        found_new, last_jump = find_next_jump_func_mapping[direction](all_jumps, last_jump)
        if not found_new:
            if current_jump == last_jump:
                return False, last_jump
            else:
                return True, last_jump


def forbid_jump_if_needed(matrix, x_mapping, y_mapping, all_jumps):
    found_next, beginning_of_chain = find_last_jump_in_direction(all_jumps[1:], all_jumps[0], Direction.BACKWARD)
    found_prev, end_of_chain       = find_last_jump_in_direction(all_jumps[1:], all_jumps[0], Direction.FORWARD)

    if not (found_prev and found_next):
        inf_x = 0
        inf_y = 0
        for i in range(matrix.shape[0]):
            if x_mapping[i] == end_of_chain.end:
                inf_x = i
            if y_mapping[i] == beginning_of_chain.begin:
                inf_y = i
        matrix[inf_x][inf_y] = POSITIVE_INF


# rethink
def generate_sub_task_data(matrix, x_mapping, y_mapping, all_jumps, zero_with_most_weight):
    new_matrix = np.empty([matrix.shape[0]-1, matrix.shape[1]-1], dtype=np.int32)
    new_x_mapping = np.empty([matrix.shape[0]-1], dtype=np.int32)
    new_y_mapping = np.empty([matrix.shape[0]-1], dtype=np.int32)
    new_all_jumps = all_jumps[:]
    new_all_jumps.insert(0, JumpType(
            x_mapping[zero_with_most_weight.row],
            y_mapping[zero_with_most_weight.column]
    ))

    i_x = 0
    i_y = 0
    for i in range(matrix.shape[0]):
        if i != zero_with_most_weight.row:
            new_x_mapping[i_x] = x_mapping[i]
            j_y = 0
            for j in range(matrix.shape[1]):
                if j == zero_with_most_weight.column:
                    continue
                new_matrix[i_x][j_y] = matrix.item(i, j)
                j_y += 1
        else:
            i_x -= 1

        if i != zero_with_most_weight.column:
            new_y_mapping[i_y] = y_mapping[i]
        else:
            i_y -= 1

        i_x += 1
        i_y += 1

    forbid_jump_if_needed(new_matrix, new_x_mapping, new_y_mapping, new_all_jumps)
    return SubTaskDataType(new_matrix, new_x_mapping, new_y_mapping, new_all_jumps)

branch_count = 0
def solve_impl(matrix, x_mapping, y_mapping, all_jumps, solution_cost, min_cost):
    global branch_count
    branch_count += 1
    if matrix.shape[0] == 0 or matrix.shape[0] == 1:
        return ERROR_ANSWER

    success, additional_cost = calculate_additional_cost_and_correct_matrix(matrix)
    if not success:
        return ERROR_ANSWER

    solution_cost += additional_cost
    zero_with_most_weight = find_zero_with_biggest_weight(matrix)

    if matrix.shape[0] == 2:
        next_city = zero_with_most_weight.column
        previous_city = zero_with_most_weight.row

        if matrix[previous_city ^ 1][next_city ^ 1] == 0 and \
           matrix[previous_city ^ 1][next_city] == POSITIVE_INF and \
           matrix[previous_city][next_city ^ 1] == POSITIVE_INF:
            final_jumps = [JumpType(
                            x_mapping[zero_with_most_weight.row],
                            y_mapping[zero_with_most_weight.column]
                        ),
                           JumpType(
                            x_mapping[zero_with_most_weight.row ^ 1],
                            y_mapping[zero_with_most_weight.column ^ 1]
                        )
            ]
            final_jumps.extend(all_jumps)
            return AnswerType(final_jumps, solution_cost)
        else:
            return ERROR_ANSWER
    # prepare data for recursive call
    new_matrix, new_x_mapping, new_y_mapping, new_all_jumps = generate_sub_task_data(
        matrix, x_mapping, y_mapping, all_jumps, zero_with_most_weight
    )

    # call this function recursively
    answer = solve_impl(new_matrix, new_x_mapping, new_y_mapping, new_all_jumps, solution_cost, min_cost)

    final_path = []
    # print_answer(answer, "GOT ANSWER");
    if answer.cost < min_cost:
        (final_path, min_cost) = answer
        # print_answer((end_of_path, min_cost), "NEW ANSWER");

    # right_path
    if solution_cost + zero_with_most_weight.weight < min_cost:
        # correct first one
        matrix[zero_with_most_weight.row][zero_with_most_weight.column] = POSITIVE_INF

        answer = solve_impl(matrix, x_mapping, y_mapping, all_jumps, solution_cost, min_cost)
        # print_answer(answer, "GOT ANSWER");
        if answer.cost < min_cost:
            (final_path, min_cost) = answer

    # return answer
    if min_cost < POSITIVE_INF:
        return AnswerType(final_path, min_cost)
    else:
        return ERROR_ANSWER


def crusher_impl(matrix, x_mapping , y_mapping, all_jumps, solution_cost, min_cost):
    success, additional_cost = calculate_additional_cost_and_correct_matrix(matrix)
    if not success:
        return False, (None, None)

    solution_cost += additional_cost
    zero_with_most_weight = find_zero_with_biggest_weight(matrix)

    # prepare data for recursive call
    new_matrix, new_x_mapping, new_y_mapping, new_all_jumps = generate_sub_task_data(
        matrix, x_mapping, y_mapping, all_jumps, zero_with_most_weight
    )

    # call this function recursively
    task1 = (new_matrix, new_x_mapping, new_y_mapping, new_all_jumps, solution_cost, min_cost)
    matrix[zero_with_most_weight.row][zero_with_most_weight.column] = POSITIVE_INF
    task2 = (matrix, x_mapping, y_mapping, all_jumps, solution_cost, min_cost)

    return True, (task1, task2)


def generate_random_task_of_size_n(n, modulus, seed):
    np.random.seed(seed)

    matrix = np.random.randint(0, modulus, [n, n])
    mapping = np.array(range(n))

    for i in range(n):
        matrix[i][i] = POSITIVE_INF

    return DataTaskType(matrix, mapping, mapping)


def test_case_task():
    matrix = np.array([
        [POSITIVE_INF,           25,           40,           31,           27],
        [           5, POSITIVE_INF,           17,           30,           25],
        [          19,           15, POSITIVE_INF,            6,            1],
        [           9,           50,           24, POSITIVE_INF,            6],
        [          22,            8,            7,           10, POSITIVE_INF]
    ])
    mapping = np.array(range(5))
    return DataTaskType(matrix, mapping, mapping)


def solve(matrix):
    mapping = np.array(range(matrix.shape[0]))
    return solve_impl(matrix, mapping, mapping, [], 0, POSITIVE_INF)


def run_test_case():
    task = test_case_task()
    t1 = time.clock()
    answer = solve_impl(task.matrix, task.column_mapping, task.row_mapping, [], 0, POSITIVE_INF)
    t2 = time.clock()
    print_answer(answer)
    print("It took {} msec\n ", t2-t1)


def run_benchmark():
    global branch_count
    for i in range(5, 25):
        branch_count = 0
        task = generate_random_task_of_size_n(i, 100, 0)
        t1 = time.clock()
        answer = solve_impl(task.matrix, task.column_mapping, task.row_mapping, [], 0, POSITIVE_INF)
        t2 = time.clock()
        print("For i : {} it took {} msec cost {} branch_count = {} ".format(i, t2-t1, answer.cost, branch_count))

if __name__ == '__main__':
    #run_test_case()
    run_benchmark()


