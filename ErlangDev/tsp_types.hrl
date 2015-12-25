-record(zero_info, {row, col, weight}).
-record(jump, {source, destination}).
-record(sub_task_data, {matrix, row_mapping, col_mapping, jumps}).
-record(answer, {jumps, cost}).
-record(task, {matrix, row_mapping, col_mapping, jumps, curr_cost, min_cost, size}).

-define(FORWARD_DIR, 1).
-define(BACKWARD_DIR, -1).
-define(POSITIVE_INF, 2147483647).
-define(NEGATIVE_INF, -2147483648).
-define(NVAL_INDEX, -1).
-define(ERROR_ANSWER, #answer{jumps = [], cost = ?POSITIVE_INF}).
-define(ERROR_TASK, #task{matrix = [], row_mapping = [], col_mapping = [], jumps = [],
    curr_cost = 0, min_cost = 0, size = 0}).