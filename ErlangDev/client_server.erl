-module(client_server).
-include("tsp_types.hrl").
-include_lib("xmerl/include/xmerl.hrl").
-import(solver, [crusher_impl/1, solve_impl/1]).
-export([test_setup/0, test_setup1/0]).

%-------------------- New connection handeler ---------------------------
start_connection_listener(WorkingQueue) ->
	spawn(fun() -> connection_listener_init(6000, WorkingQueue) end) ! {status, self()},
	receive
		ok ->
	  		io:fwrite("[CONNECTION_LISTENER~w] Listening on port ~p~n", [self(), 6000]);
		fail ->
	  		io:fwrite("[CONNECTION_LISTENER~w] Internal error: unable to start listener~n", [self()])
	end.

connection_listener_init(Port, WorkingQueue) ->
	receive {status, ServerPid} ->
		try gen_tcp:listen(Port, [binary, {packet, 0}, {reuseaddr, true}, {active, false}, {ip, {0,0,0,0}}]) of
  			{ok, Socket} ->
	    		ServerPid ! ok,
	    		connection_listener_loop(Socket, WorkingQueue);
	  		_ ->
	    		ServerPid ! fail
		catch
	  		_:_ ->
	    	ServerPid ! fail
		end
	end.

connection_listener_loop(ListenSocket, WorkingQueue) ->
	try gen_tcp:accept(ListenSocket) of
		{ok, ClientSocket} ->
  			io:fwrite("[CONNECTION_LISTENER~w] New connection [~p] accepted~n", [ClientSocket, spawn(fun() -> new_connection_handler(ClientSocket, WorkingQueue) end)]);
		_ ->
  			io:fwrite("[CONNECTION_LISTENER~w] Internal error: Unable to accept a connection~n", [self()])
	catch
		_:_ ->
  			io:fwrite("[CONNECTION_LISTENER~w] Internal error: Unable to accept a connection~n", [self()])
	end,
	connection_listener_loop(ListenSocket, WorkingQueue).
%-------------------- New connection handeler ---------------------------
task_handler(Task, MinValue, NumberOfGeneratedTasks, NumberOfSolvedTasks, FinishedGenerationgTasks) ->
	io:fwrite("[TASK HANDLER~w] Waiting for event. NumberOfGeneratedTasks: ~w NumberOfSolvedTasks: ~w FinishedGenerationgTasks: ~w ~n", [self(), NumberOfGeneratedTasks, NumberOfSolvedTasks, FinishedGenerationgTasks]),
	receive
		{new_task_request, TaskQueuePID} ->
			io:fwrite("[TASK HANDLER~w] New task request recieved. Updating TaskInfo...~n", [self()]),
			% Generate task
			UpdatedTask = #task{
								matrix = Task#task.matrix, 
								row_mapping = Task#task.row_mapping, 
								col_mapping = Task#task.col_mapping,
    							jumps = Task#task.jumps, 
    							curr_cost = Task#task.curr_cost, 
    							min_cost = MinValue#answer.cost, 
    							size = Task#task.size
    							},
			io:fwrite("[TASK HANDLER~w] Updating done. Calling crusher_impl...~n", [self()]),
			{Success, SubTask, RestTasks} = crusher_impl(UpdatedTask),
			io:fwrite("[TASK HANDLER~w] Crasher_impl done.~n", [self()]),
			% Send task to queueu
			if
				Success =:= true ->
					io:fwrite("[TASK HANDLER~w] Crasher_impl return success. Sending ok message...~n", [self()]),
					TaskQueuePID ! {ok, {self(), SubTask}},
					io:fwrite("[TASK HANDLER~w] Sending of ok message done. Sending of new_task message...~n", [self()]),
					TaskQueuePID ! {new_task, self()},
					io:fwrite("[TASK HANDLER~w] Sending of new_task message done.~n", [self()]),
					task_handler(RestTasks, MinValue, NumberOfGeneratedTasks+1, NumberOfSolvedTasks, FinishedGenerationgTasks);
				true ->
					io:fwrite("[TASK HANDLER~w] Crusher_impl failed to generate new task. Sending failed message...~n", [self()]),
					TaskQueuePID ! {failed},
					io:fwrite("[TASK HANDLER~w] Sending of failed message done~n", [self()]),
					if 
						NumberOfSolvedTasks =:= NumberOfGeneratedTasks ->
							io:fwrite("[TASK HANDLER~w] Task solved. Starting to handle it...~n", [self()]),
							MinValue;
						true ->
							task_handler(RestTasks, MinValue, NumberOfGeneratedTasks, NumberOfSolvedTasks, true)
					end
			end;
		{handle_new_answer, NewAnswer} ->
			io:fwrite("[TASK HANDLER~w] New answer arrived. Starting to handle it ~w...~n", [self(), NewAnswer#answer.cost]),
			if
				MinValue#answer.cost > NewAnswer#answer.cost ->
            		NewMinValue = NewAnswer;
        		true -> 
        			NewMinValue = MinValue
    		end,
			if 
				(FinishedGenerationgTasks =:= true) and 
				((NumberOfSolvedTasks =:= NumberOfGeneratedTasks) or (NumberOfSolvedTasks + 1 =:= NumberOfGeneratedTasks)) ->
					io:fwrite("[TASK HANDLER~w] Task solved. Starting to handle it...~n", [self()]),
					NewMinValue;
				true ->
					io:fwrite("[TASK HANDLER~w] Although subtask solved. Task not solved.~n", [self()]),
					task_handler(Task, NewMinValue, NumberOfGeneratedTasks, NumberOfSolvedTasks+1, FinishedGenerationgTasks)		
			end
	end.

new_connection_handler(ClientSocket, WorkingQueue) ->
	io:fwrite("[NEW CONNECTION HANDLER~w] Waiting for message size...~n", [self()]),
	case gen_tcp:recv(ClientSocket, 8) of
		{ok, SizeData} ->
			<< MessageSize:8/little-signed-integer-unit:8>> = SizeData
	end,
	io:fwrite("[NEW CONNECTION HANDLER~w] Waiting for message...~n", [self()]),
	case gen_tcp:recv(ClientSocket, MessageSize) of
		{ok, XMLData} ->
			% Deserialize data.
			io:fwrite("[NEW CONNECTION HANDLER~w] Deserialize task...~n", [self()]),
			Task = deserialize_task(XMLData),
			io:fwrite("[NEW CONNECTION HANDLER~w] Deserialize task is :~n Matris:~n~w~nSize: ~w~n", [self(), Task#task.matrix, Task#task.size]),
			% Notify task queue that we have new tasks.
			io:fwrite("[NEW CONNECTION HANDLER~w] Send that we are ready to give tasks...~n", [self()]),
			WorkingQueue ! {new_task, self()},
			% Solve.
			io:fwrite("[NEW CONNECTION HANDLER~w] Solve...~n", [self()]),
			Answer = task_handler(Task, #answer{jumps = [], cost = ?POSITIVE_INF}, 0, 0, false),
			% Serialize answer and send it back.
			io:fwrite("[NEW CONNECTION HANDLER~w] Serializing... ~w~n", [self(), Answer]),
			[AnswerBinDataSizeBinary, AnswerBinDataSize] = serialize_answer(Answer), 
			io:fwrite("[NEW CONNECTION HANDLER~w] Sending...~n", [self()]),
			gen_tcp:send(ClientSocket, AnswerBinDataSizeBinary),
			gen_tcp:send(ClientSocket, AnswerBinDataSize),
			io:fwrite("[NEW CONNECTION HANDLER~w] Done~n", [self()])
	end.

deserialize_task(XMLData)->
	XMLStr = unicode:characters_to_list(XMLData, utf8),
	{Xml, _} = xmerl_scan:string(XMLStr),
	% Parse int
	{_, SizeBinary} = val(xmerl_xpath:string("//size", Xml)),
	SizeString = unicode:characters_to_list(SizeBinary, utf8),
	{Size, _} = string:to_integer(SizeString),
	io:fwrite("[NEW CONNECTION HANDLER~w] Size:~w~n", [self(), Size]),
	% Parse matrix
	{_, MatrixBinary} = val(xmerl_xpath:string("//matrix", Xml)),
	MatrixString = unicode:characters_to_list(MatrixBinary),
	MatrixLinear = [ begin {Int,_} = string:to_integer(Token), Int end || Token<-string:tokens(MatrixString," ")],
	Matrix = split_matrix(Size, Size, MatrixLinear),
	io:fwrite("[NEW CONNECTION HANDLER~w] Matrix:~w~n", [self(), Matrix]),
	Mapping = lists:seq(0, Size - 1),
    Task = #task{matrix = Matrix, row_mapping = Mapping, col_mapping = Mapping, jumps = [],
    			curr_cost = 0, min_cost = ?POSITIVE_INF, size = Size}.

split_row_impl(0, List, Accumulator) ->
	{List, Accumulator};

split_row_impl(Size, [Element | Rest], Accumulator) ->
	split_row_impl(Size-1, Rest, Accumulator ++ [Element]).

split_row(Size, List) ->
	split_row_impl(Size, List, []).

split_matrix_impl(0, ColumnSize, [], Accumulator)->
	Accumulator;

split_matrix_impl(RowSize, ColumnSize, Matrix, Accumulator)->
	{RestMatrix, Row} = split_row(ColumnSize, Matrix),
	split_matrix_impl(RowSize - 1, ColumnSize, RestMatrix, Accumulator ++ [Row]).

split_matrix(RowSize, ColumnSize, Matrix)->
	split_matrix_impl(RowSize, ColumnSize, Matrix, []).

serialize_answer(Answer)->
	AnswerBinData = creatXml(Answer),
	AnswerBinDataSize = length(AnswerBinData),
	AnswerBinDataSizeBinary = << AnswerBinDataSize:8/little-signed-integer-unit:8 >>,
	[AnswerBinDataSizeBinary, AnswerBinData].

val(X) ->
	[#xmlElement{name = N, content = [#xmlText{value = V}|_]}] = X,
	{N, V}.

creatXml(Answer)->
  	RootElem = {array, [
					{cost, [integer_to_list(Answer#answer.cost)]}, 
					{jumps, [serialize_jumps(Answer#answer.jumps)]}
				]},
  	lists:flatten(xmerl:export_simple([RootElem], xmerl_xml)).

serialize_rest_jumps([], Accumulator) ->
  	Accumulator;

serialize_rest_jumps([Jump | Tail], Accumulator) ->
	SubString = "," ++ integer_to_list(Jump#jump.source) ++ "-" ++ integer_to_list(Jump#jump.destination),
	serialize_rest_jumps(Tail, Accumulator ++ SubString).

serialize_jumps([]) ->
	"";

serialize_jumps([Jump | Tail]) ->
	SubString = integer_to_list(Jump#jump.source) ++ "-" ++ integer_to_list(Jump#jump.destination),
	serialize_rest_jumps(Tail, SubString).
%-------------------- Worker Queue ---------------------------
task_queue(State, FreeWorkerQueue) ->
	if 
		State == waiting_for_generator ->
			io:fwrite("[TASK QUEUE~w] Waiting for generator~n", [self()]),
			receive
				{new_task, TaskGeneratorPID} ->
					io:fwrite("[TASK QUEUE~w] Got new generator. Sending request back...~n", [self()]),
					TaskGeneratorPID ! {new_task_request, self()},
					io:fwrite("[TASK QUEUE~w] Sending request back done.~n", [self()])
			end,
			io:fwrite("[TASK QUEUE~w] Switching state to sending_request~n", [self()]),
			task_queue(sending_request, FreeWorkerQueue);

		State == sending_request ->
			io:fwrite("[TASK QUEUE~w] Waiting for task...~n", [self()]),
			receive
				{ok, {TaskHandlerPID, Task}} ->
					io:fwrite("[TASK QUEUE~w] New task successfully acuired. Sending request for free worker...~n", [self()]),
					FreeWorkerQueue ! {request, self()},
					receive
						{free_worker, WorkerPID} ->
							io:fwrite("[TASK QUEUE~w] Got free worker. Sending him task...~n", [self()]),
							WorkerPID ! {do_task, {TaskHandlerPID, Task}},
							io:fwrite("[TASK QUEUE~w] Done sending task.~n", [self()])
					end;
				{failed} ->
					io:fwrite("[TASK QUEUE~w] Generator exhosted~n", [self()])
			end,
			io:fwrite("[TASK QUEUE~w] Switching state to waiting_for_generator~n", [self()]),
			task_queue(waiting_for_generator, FreeWorkerQueue)
	end.
%-------------------- FreeWorkerQueue ---------------------------
free_worker_queue() ->
	io:fwrite("[FREE WORKER QUEUE~w] Waiting for free worker request~n", [self()]),
	receive
		{request, ReceiverPID} -> 
			io:fwrite("[FREE WORKER QUEUE~w] Got new request for free worker~n", [self()]),
			receive
				{new_free_worker, WorkerPID} ->
					io:fwrite("[FREE WORKER QUEUE~w] Sending free warker to ReceiverPID~n", [self()]),
					ReceiverPID ! {free_worker, WorkerPID},
					io:fwrite("[FREE WORKER QUEUE~w] Done~n", [self()])
			end
	end,
	free_worker_queue().
%-------------------- Worker ---------------------------
worker(FreeWorkerQueue) ->
	io:fwrite("[WORKER~w] Sending to FreeWorkerQueue that i'm free~n", [self()]),
	FreeWorkerQueue ! {new_free_worker, self()},
	io:fwrite("[WORKER~w] Waiting for task~n", [self()]),
	receive
		{do_task, {TaskHandlerPID, Task}} ->
			io:fwrite("[WORKER~w] Got new task, solving it...~n", [self()]),
			Answer = solve_impl(Task),
			io:fwrite("[WORKER~w] Done new task, sending answer ~w...~n", [self(), Answer#answer.cost]),
			TaskHandlerPID ! {handle_new_answer, Answer}
	end,
	io:fwrite("[WORKER~w] Done sending answer~n", [self()]),
	worker(FreeWorkerQueue).
%-------------------- Test Setup ---------------------------	
test_setup() -> 
	FreeWorkerQueuePID = spawn(fun() -> free_worker_queue() end),
	TaskQueuePID = spawn(fun() -> task_queue(waiting_for_generator, FreeWorkerQueuePID) end),
	io:format("[INITIALIZATION COMPLETED]~n"),
	spawn(fun()-> start_connection_listener(TaskQueuePID) end),
	WorkerPID = spawn(fun() -> worker(FreeWorkerQueuePID) end).
	%{ok, [X]} = io:fread("How many Hellos?> ", "~d").

test_setup1() -> 
	start_connection_listener([]).

























