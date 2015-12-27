-module(solver).
-include("tsp_types.hrl").
-export([test1/0, crusher_impl/1, solve_impl/1]).

%--------------------------------------------------------

print_list([]) -> io:fwrite("\n");
print_list([?POSITIVE_INF | Rest]) ->
    io:fwrite(" INF"),
    print_list(Rest);
print_list([Elem | Rest]) ->
    io:fwrite(" ~w",[Elem]),
    print_list(Rest).

print_matrix([]) -> io:fwrite("\n");
print_matrix([Row | Rest]) ->
    print_list(Row),
    print_matrix(Rest).

%----------------------- General Functions ------------------ 
transpose([[]|_]) -> [];
transpose(M) ->
  [lists:map(fun hd/1, M) | transpose(lists:map(fun tl/1, M))].

nthtail(Index, []) -> [];
nthtail(0, L) -> L;
nthtail(Index, [ _ | Tail]) -> nthtail(Index-1, Tail).

replace_list_element([], Head, 0, 0, Value) -> Head ++ [Value];
replace_list_element([], Head, CurrIndex, Index, Value) -> Head;
replace_list_element([El | Tail], [], Index, Index, Value) -> [Value] ++ Tail;
replace_list_element([El | Tail], [], CurrIndex, Index, Value) ->
    replace_list_element(Tail, [El], CurrIndex+1, Index, Value);
replace_list_element([El | Tail], Head, Index, Index, Value) -> Head ++ [Value] ++ Tail;
replace_list_element([El | Tail], Head, CurrIndex, Index, Value) ->
    replace_list_element(Tail, Head++[El], CurrIndex+1, Index, Value).

replace_list_element(List, Index, Value) ->
    replace_list_element(List, [], 0, Index, Value).
%replace_list_element(List, Index, Value) ->
%    lists:sublist(List, Index) ++ [Value] ++ lists:nthtail(Index+1, List).

remove_list_element(List, Index) ->
    lists:sublist(List, Index) ++ lists:nthtail(Index+1, List).

nth( _, _, []) -> [];
nth(Index, Index, [Elem | Rest]) -> Elem;
nth(CurrIndex, Index, [Elem | Rest]) -> nth(CurrIndex+1, Index, Rest).

nth(Index, L) -> nth(0, Index, L).

set_to_matrix(M, RowIndex, ColIndex, Value) ->
    Row = nth(RowIndex, M),
    NewRow = replace_list_element(Row, ColIndex, Value),
    replace_list_element(M, RowIndex, NewRow).

get_from_matrix(M, RowIndex, ColIndex) ->
    lists:nth(ColIndex+1, lists:nth(RowIndex+1, M)).

update_task_matrix(T, RowIndex, ColIndex, Value) ->
    NewMatrix = set_to_matrix(T#task.matrix, RowIndex, ColIndex, Value),
    #task{matrix = NewMatrix, row_mapping = T#task.row_mapping, col_mapping = T#task.col_mapping,
        jumps = T#task.jumps, curr_cost = T#task.curr_cost, min_cost = T#task.min_cost, size = T#task.size}.

update_task_min_cost(T, Value) ->
    #task{matrix = T#task.matrix, row_mapping = T#task.row_mapping, col_mapping = T#task.col_mapping,
        jumps = T#task.jumps, curr_cost = T#task.curr_cost, min_cost = Value, size = T#task.size}.

map(Fun, Acc, [El | Rest]) ->
    [Fun(El, Acc) | map(Fun, Acc+1, Rest)];
map(_,_,[]) -> [].

foreach(Fun, Acc, [El | Rest]) ->
    Fun(El, Acc), foreach(Fun, Acc+1, Rest);
foreach(_,Acc,[]) -> Acc.

foreach(Fun1, Acc1, Fun2, Acc2, [El | Rest]) ->
    foreach(Fun1, Acc1+1, Fun2, Fun2(Fun1(El, Acc1), Acc2), Rest);
foreach(_,_,_,Acc2,[]) -> Acc2.

foreach(Fun1, X, Y, Fun2, Acc1, Acc2, [El | Rest]) ->
    {NX, NY} = Fun1(El, X, Y),
    {NAcc1, NAcc2} = Fun2(NX, NY, Acc1, Acc2),
    foreach(Fun1, NX, NY, Fun2, NAcc1, NAcc2, Rest);
foreach(_,_,_,_,Acc1,Acc2,[]) -> {Acc1, Acc2}.

foreach(Fun1, X, Y, Fun2, Acc, [El | Rest]) ->
    {NX, NY} = Fun1(El, X, Y),
    foreach(Fun1, NX, NY, Fun2, Fun2(NX, NY, Acc), Rest);
foreach(_,_,_,_,Acc,[]) -> Acc.

%------------------ calculate_plus_cost -----------------------

cpc_min_value([], Counter, _ , Counter) -> {false, 0, 0};
cpc_min_value([], Counter, MV , InfCount ) -> {true, MV, InfCount};
cpc_min_value([?POSITIVE_INF | Rest], Counter, MV, InfCount) ->
    cpc_min_value(Rest, Counter+1, MV, InfCount+1);
cpc_min_value([Elem | Rest], Counter, MV, InfCount) when Elem < MV ->
    cpc_min_value(Rest, Counter+1, Elem, InfCount);
cpc_min_value([ _ | Rest], Counter, MV, InfCount) ->
    cpc_min_value(Rest, Counter+1, MV, InfCount).

cpc_update_line(L, _ , 0) -> L;
cpc_update_line([], NewL , _) -> NewL;
cpc_update_line([?POSITIVE_INF | Rest], [], MV) -> cpc_update_line(Rest, [?POSITIVE_INF], MV);
cpc_update_line([Elem | Rest], [], MV) -> cpc_update_line(Rest, [(Elem-MV)], MV);
cpc_update_line([?POSITIVE_INF | Rest], NewL, MV) -> cpc_update_line(Rest, NewL++[?POSITIVE_INF], MV);
cpc_update_line([Elem | Rest], NewL, MV) -> cpc_update_line(Rest, NewL++[(Elem-MV)], MV).

cpc_cost_for_line(true, [], NewMatrix, _, _ , Cost) -> {true, NewMatrix, Cost};
cpc_cost_for_line(false, _, NewMatrix, _, _, Cost) -> 
    %io:fwrite("[cadasd] yuo asdasd: ~w\n",[Cost]), 
    {false, NewMatrix, Cost};
cpc_cost_for_line(true, [Elem | Rest], NewMatrix, ElemIndex, MV, Cost) ->
    {IsError, NewMV, NewInfCount} = cpc_min_value(Elem, 0, ?POSITIVE_INF, 0),
    NewRow = cpc_update_line(Elem, [], NewMV),
    %io:fwrite("[cpc_cost_for_line] calc 1: ~w\n", [NewMatrix]),
    NewMatrix2 = replace_list_element(NewMatrix, ElemIndex, NewRow),
    %io:fwrite("[cpc_cost_for_line] calc 2: ~w\n", [NewMatrix2]),
    cpc_cost_for_line(IsError, Rest, NewMatrix2, ElemIndex+1, NewMV, Cost+NewMV).

calculate_plus_cost(Matrix) ->
    {Success, M1, Cost1} = cpc_cost_for_line(true, Matrix, Matrix, 0, ?POSITIVE_INF, 0),
    %io:fwrite("M1:\n ~w \n", [M1]),
    if
        Success =:= false -> {false, [], 0};
        true ->
            %io:fwrite("[calculate_plus_cost] Transpose: ~w\n", [M1]),
            Columns = transpose(M1),
            %io:fwrite("[calculate_plus_cost] Transposes: ~w\n", [Columns]),
            {Success2, M2, Cost2} = cpc_cost_for_line(true, Columns, Columns, 0, ?POSITIVE_INF, Cost1),
            %io:fwrite("[calculate_plus_cost] pass calc 1: ~w\n", [M2]),
            M3 = transpose(M2),
            %io:fwrite("[calculate_plus_cost] pass calc 2\n"),
            {Success2, M3, Cost2}
    end.

%---------------- find_heaviest_zero --------------------------

fhz_update_weight(W, ?POSITIVE_INF) -> W;
fhz_update_weight(W, Value) -> W + Value.

find_heaviest_zero(Matrix) ->
    Columns = transpose(Matrix),
    foreach(fun(Row, RowIndex)->
        foreach(fun(Col, ColIndex)->
                if
                    Col == 0 ->
                        W = fhz_update_weight(fhz_update_weight(0, lists:min(remove_list_element(Row, ColIndex))),
                        lists:min(remove_list_element(lists:nth(ColIndex+1, Columns), RowIndex))),
                        #zero_info{row = RowIndex, col = ColIndex, weight = W};
                    true -> #zero_info{row = RowIndex, col = ColIndex, weight = ?NEGATIVE_INF}
                end
            end, 0, fun(W, HZ)->
                if
                    HZ#zero_info.weight < W#zero_info.weight -> W;
                    true -> HZ
                end
            end, #zero_info{row = 0, col = 0, weight = ?NEGATIVE_INF}, Row) end,
    0, fun(W, HZ)->
        if
            HZ#zero_info.weight < W#zero_info.weight -> W;
            true -> HZ
        end
    end, #zero_info{row = 0, col = 0, weight = ?NEGATIVE_INF}, Matrix).

%-------------------------- find_previous_jump -----------------------------

find_previous_jump([], CurrJump) -> {CurrJump, false};
find_previous_jump([Jump | Rest], CurrJump) ->
    if
        Jump#jump.destination =:= CurrJump#jump.source -> {Jump, true};
        true -> find_previous_jump(Rest, CurrJump)
    end.

%---------------------------- find_next_jump -----------------------------

find_next_jump([], CurrJump) -> {CurrJump, false};
find_next_jump([Jump | Rest], CurrJump) ->
    if
        Jump#jump.source =:= CurrJump#jump.destination -> {Jump, true};
        true -> find_next_jump(Rest, CurrJump)
    end.

%-------------------- find_last_jump_in_direction ---------------------------

flj_choose_direction(?FORWARD_DIR, AllJumps, LastJump) -> find_next_jump(AllJumps, LastJump);
flj_choose_direction(?BACKWARD_DIR, AllJumps, LastJump) -> find_previous_jump(AllJumps, LastJump);
flj_choose_direction( _ , _ , _ ) ->
    io:fwrite("[find_last_jump_in_direction] Error: No right direction!"),
    {#jump{source = -1, destination = -1}, false}.

flj_loop_search(false, _ , _ , LastJump) -> {LastJump, false};
flj_loop_search(true, Dir, AllJumps, LastJump) ->
    {FindLastJump, FoundNew} = flj_choose_direction(Dir, AllJumps, LastJump),
    flj_loop_search(FoundNew, Dir, AllJumps, FindLastJump).

find_last_jump_in_direction(AllJumps, CurrJump, _) ->
    {LastJump, _ } = flj_loop_search(true, ?FORWARD_DIR, AllJumps, CurrJump),
    if
        LastJump#jump.source < 0 -> {LastJump, false};
        (CurrJump#jump.source =/= LastJump#jump.source) and (CurrJump#jump.destination =/= LastJump#jump.destination) ->
            {LastJump, true};
        true -> {LastJump, false}
    end.

%------------------- forbid_jump_if_needed -----------------------------

find_inf_pos([], _ , InfIndex, _ ) -> InfIndex;
find_inf_pos([CheckedValue | Rest], Index, _ , CheckedValue) ->
    find_inf_pos(Rest, Index+1, Index, CheckedValue);
find_inf_pos([ _ | Rest], Index, InfIndex, CheckedValue) ->
    find_inf_pos(Rest, Index+1, InfIndex, CheckedValue).

forbid_jump_if_needed(M, RowMapping, ColMapping, [J | JumpsRest]) ->
    {BC, FoundNext} = find_last_jump_in_direction(JumpsRest, J, ?BACKWARD_DIR),
    {EC, FoundPrev} = find_last_jump_in_direction(JumpsRest, J, ?FORWARD_DIR),
    %io:fwrite("solve_impl 3.1.1: ~w ", [BC]),
    %io:fwrite(" ~w | ", [FoundNext]),
    %io:fwrite("~w", [EC]),
    %io:fwrite(" ~w\n", [FoundPrev]),
    if 
        not (FoundPrev and FoundNext) ->
            %io:fwrite("cond 1\n"),
            RowIndex = find_inf_pos(RowMapping, 0, 0, EC#jump.destination),
            ColIndex = find_inf_pos(ColMapping, 0, 0, BC#jump.source),
            %io:fwrite("row = ~w, ",[RowIndex]),
            %io:fwrite("col = ~w\n", [ColIndex]),
            set_to_matrix(M, RowIndex, ColIndex, ?POSITIVE_INF);
        true -> 
            %io:fwrite("cond 2\n"), 
            M
    end.

%----------------- generate_sub_task_data --------------------

generate_sub_task_data(Matrix, RowMapping, ColMapping, AllJumps, HeaviestZero) ->
    ZeroRow = HeaviestZero#zero_info.row,
    ZeroCol = HeaviestZero#zero_info.col,
    %io:fwrite("Zero: ~w ", [ZeroRow]),
    %io:fwrite("~w\n", [ZeroCol]),
    NewAllJumps = [#jump{source = lists:nth(ZeroRow+1, RowMapping), destination = lists:nth(ZeroCol+1, ColMapping)}] ++ AllJumps,
    NewMatrix = lists:map(fun(Row) -> remove_list_element(Row, ZeroCol) end, remove_list_element(Matrix, ZeroRow)),
    NewRowMapping = remove_list_element(RowMapping, ZeroRow),
    NewColMapping = remove_list_element(ColMapping, ZeroCol),
    %io:fwrite("solve_impl 3.1\n"),
    %print_matrix(NewMatrix),
    M2 = forbid_jump_if_needed(NewMatrix, NewRowMapping, NewColMapping, NewAllJumps),
    %io:fwrite("solve_impl 3.2\n"),
    %print_matrix(M2),
    #sub_task_data{matrix = M2, row_mapping = NewRowMapping,
        col_mapping = NewColMapping, jumps = NewAllJumps}.

%------------------- solve_impl -----------------------------
solve_update_path( _ , A, T) when A#answer.cost < T#task.min_cost -> T1 = update_task_min_cost(T, A#answer.cost), {A#answer.jumps, T1};
solve_update_path(CurrPath, _, T) -> {CurrPath, T}.

solve_right_path(T, HZ, FinalPath) when (T#task.curr_cost + HZ#zero_info.weight) < T#task.min_cost ->
    %io:fwrite("dfsdfsdf: ~w\n", [T#task.matrix]),
    T1 = update_task_matrix(T, HZ#zero_info.row, HZ#zero_info.col, ?POSITIVE_INF),
    A = solve_impl(T1#task.size, T1),
    solve_update_path(FinalPath, A, T1);
solve_right_path(T, HZ, FinalPath) -> {FinalPath, T}.

solve_impl(T) -> solve_impl(T#task.size, T).

solve_impl(0, T) -> ?ERROR_ANSWER;
solve_impl(1, T) -> ?ERROR_ANSWER;
solve_impl(Size, T) ->
    %io:fwrite("solve_impl 1\n"),
    %file:write(IODevice, "solve_impl 1\n"),
    %print_matrix(T#task.matrix),
    {Success, NewMatrix, PlusCost} = calculate_plus_cost(T#task.matrix),
    %io:fwrite("pass 1\n"),
    if
        not Success -> ?ERROR_ANSWER;
        true ->
            T1 = #task{matrix = NewMatrix, row_mapping = T#task.row_mapping, col_mapping = T#task.col_mapping, jumps = T#task.jumps,
                curr_cost = T#task.curr_cost + PlusCost, min_cost = T#task.min_cost, size = T#task.size},
            HZ = find_heaviest_zero(NewMatrix),
            %io:fwrite("pass 2\n"),
            %io:fwrite("solve_impl 2 ~w\n",[T1#task.matrix]),
            %io:fwrite("solve_impl 2\n"),
            %print_matrix(T1#task.matrix),
            solve_impl(Size, HZ, T1)
    end.
solve_impl(2, HZ, T) ->
    %io:fwrite("solve_impl size == 2\n"),
    %io:fwrite("solve_impl size == 2\n"),
    NextCity = HZ#zero_info.col,
    PreviousCity = HZ#zero_info.row,
    Elem1 = get_from_matrix(T#task.matrix, PreviousCity bxor 1, NextCity bxor 1),
    Elem2 = get_from_matrix(T#task.matrix, PreviousCity bxor 1, NextCity),
    Elem3 = get_from_matrix(T#task.matrix, PreviousCity, NextCity bxor 1),
    if
        (Elem1 == 0) and (Elem2 == ?POSITIVE_INF) and (Elem3 == ?POSITIVE_INF) ->
            J1 = #jump{source = lists:nth(HZ#zero_info.row+1, T#task.row_mapping),
                    destination = lists:nth(HZ#zero_info.col+1, T#task.col_mapping)},
            J2 = #jump{source = lists:nth((HZ#zero_info.row bxor 1) + 1, T#task.row_mapping),
                    destination = lists:nth((HZ#zero_info.col bxor 1) + 1, T#task.col_mapping)},
            NewJumps = [J1, J2]++T#task.jumps,
            #answer{jumps = NewJumps, cost = T#task.curr_cost};
        true -> ?ERROR_ANSWER
    end;
solve_impl(Size, HZ, T)->
    %io:fwrite("solve_impl 3\n"),
    %print_matrix(T#task.matrix),
    %io:fwrite("solve_impl 3(1) size == ~w\n", [Size]),
    SubDT = generate_sub_task_data(T#task.matrix, T#task.row_mapping, T#task.col_mapping, T#task.jumps, HZ),
    T1 = #task{matrix = SubDT#sub_task_data.matrix, row_mapping = SubDT#sub_task_data.row_mapping, col_mapping = SubDT#sub_task_data.col_mapping,
        jumps = SubDT#sub_task_data.jumps, curr_cost = T#task.curr_cost, min_cost = T#task.min_cost, size = T#task.size - 1},
    A = solve_impl(T1),
    {FinalPath, T2} = solve_update_path([], A, T),
    %io:fwrite("solve_impl 4 ~w\n",[T2]),
    %io:fwrite("solve_impl 4\n"),
    %print_matrix(T2#task.matrix),
    {FinalPath2, T3} = solve_right_path(T2, HZ, FinalPath),
    %io:fwrite("solve_impl 5\n"),
    %print_matrix(T3#task.matrix),
    %io:fwrite("solve_impl 5 ~w\n",[T3]),
    if
        T3#task.min_cost < ?POSITIVE_INF ->
            #answer{jumps = FinalPath2, cost = T3#task.min_cost};
        true -> ?ERROR_ANSWER
    end.

%------------------- crusher_impl -----------------------------
crusher_impl(T) ->
    {Success, NewMatrix, PlusCost} = calculate_plus_cost(T#task.matrix),
    if
        Success =:= false -> {false, ?ERROR_TASK, ?ERROR_TASK};
        true ->
            HZ = find_heaviest_zero(NewMatrix),
            SubDT = generate_sub_task_data(NewMatrix, T#task.row_mapping, T#task.col_mapping, T#task.jumps, HZ),
            %io:fwrite("solve_impl 3 ~w\n", [SubDT#sub_task_data.matrix]),
            %io:fwrite("solve_impl 3\n"),
            %print_matrix(SubDT#sub_task_data.matrix),
            {
                true,
                #task{matrix = SubDT#sub_task_data.matrix, row_mapping = SubDT#sub_task_data.row_mapping, col_mapping = SubDT#sub_task_data.col_mapping,
                jumps = SubDT#sub_task_data.jumps, curr_cost = T#task.curr_cost + PlusCost, min_cost = T#task.min_cost, size = T#task.size - 1},
                update_task_matrix(T, HZ#zero_info.row, HZ#zero_info.col, ?POSITIVE_INF)
            }
    end.

%-------------test1 ------------

test1() ->
    io:fwrite("Start\n"),
    Matrix = [
    [?POSITIVE_INF,            25,            40,            31,            27],
    [            5, ?POSITIVE_INF,            17,            30,            25],
    [           19,            15, ?POSITIVE_INF,             6,             1],
    [            9,            50,            24, ?POSITIVE_INF,             6],
    [           22,             8,             7,            10, ?POSITIVE_INF]],
    Mapping = lists:seq(0, 4),
    %find_heaviest_zero(Matrix).
    T = #task{matrix = Matrix, row_mapping = Mapping, col_mapping = Mapping, jumps = [],
        curr_cost = 0, min_cost = ?POSITIVE_INF, size = 5},
    A = solve_impl(T),
    A.
    %io:fwrite("Before:\n ~w \n", [Matrix]),
    %{_, M2, Cost} = calculate_plus_cost(Matrix),
    %io:fwrite("Finish \n"),
    %io:fwrite("Cost: ~w\n", [Cost]),
    %io:fwrite("After: ~w\n", [M2]).
    %A.
    %io:fwrite("Cost = ~w, jumps = ~w \n", A#answer.cost, A#answer.jumps).