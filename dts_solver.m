dts_solver: module {
    init1: fn();
    init: fn(nil: ref Draw->Context, argv: list of string);
    solve: fn(matrix: MatrixType) : AnswerType;
    generate_random_task_of_size_n: fn(n: IndexType, modulus: int, seed: int) : DataTaskType;
    crusher_impl: fn(matrix : MatrixType, x_mapping : array of IndexType, y_mapping : array of IndexType, all_jumps: list of JumpType, solution_cost : DataType, min_cost : DataType, n : IndexType): (int, (TaskType, TaskType));
    solve_impl: fn(matrix : MatrixType, x_mapping : array of IndexType, y_mapping : array of IndexType, all_jumps: list of JumpType, solution_cost : DataType, min_cost : DataType, n : IndexType): AnswerType;


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
    TaskType : type (MatrixType, array of IndexType, array of IndexType, list of JumpType, DataType, DataType, IndexType);
    # custom constants
    ERROR_TASK : con (array[0] of DataType, array[0] of IndexType, array[0] of IndexType, nil, 0, 0, 0);
    FORWARD_DIR : con 1;
    BACKWARD_DIR : con -1;
    POSITIVE_INF : con 2147483647;
    NEGATIVE_INF : con -2147483647;
    NVAL_INDEX : con -1;
    ERROR_ANSWER : con (nil, POSITIVE_INF);
    False : con 0;
    True  : con 1;
};