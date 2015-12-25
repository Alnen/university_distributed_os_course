-module(client_server).
-include("tsp_types.hrl").
-import(solver, [crusher_impl/1, solve_impl/1]).
-export([test_setup/0]).

%-------------------- New connection handeler ---------------------------
start_connection_listener(Port, WorkingQueue) ->
	spawn(fun() -> connection_listener_init(Port, WorkingQueue) end) ! {status, self()},
	receive
		ok ->
	  		io:fwrite("[CONNECTION_LISTENER~w] Listening on port ~p~n", [self(), Port]);
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
  			io:fwrite("[CONNECTION_LISTENER~w] Internal error: Unable to accept a connection~n")
	catch
		_:_ ->
  			io:fwrite("[CONNECTION_LISTENER~w] Internal error: Unable to accept a connection~n")
	end,
	connection_listener_loop(ListenSocket, WorkingQueue).
%-------------------- New connection handeler ---------------------------
task_handler(Task, MinValue, NumberOfGeneratedTasks, NumberOfSolvedTasks, FinishedGenerationgTasks) ->
	io:fwrite("[TASK HANDLER~w] Waiting for event~n", [self()]),
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
				(FinishedGenerationgTasks =:= true) and (NumberOfSolvedTasks =:= NumberOfGeneratedTasks) ->
					io:fwrite("[TASK HANDLER~w] Task solved. Starting to handle it...~n", [self()]),
					NewMinValue;
				true ->
					io:fwrite("[TASK HANDLER~w] Although subtask solved. Task not solved.~n", [self()]),
					task_handler(Task, NewMinValue, NumberOfGeneratedTasks, NumberOfSolvedTasks+1, FinishedGenerationgTasks)		
			end
	end.

new_connection_handler_test(ClientSocket, WorkingQueue) ->
	%Deserialize
	WorkingQueue ! {new_task, self()},
	Matrix = [
    [?POSITIVE_INF,            25,            40,            31,            27],
    [            5, ?POSITIVE_INF,            17,            30,            25],
    [           19,            15, ?POSITIVE_INF,             6,             1],
    [            9,            50,            24, ?POSITIVE_INF,             6],
    [           22,             8,             7,            10, ?POSITIVE_INF]],
    Mapping = lists:seq(0, 4),
    %find_heaviest_zero(Matrix).
    Task = #task{matrix = Matrix, row_mapping = Mapping, col_mapping = Mapping, jumps = [],
        curr_cost = 0, min_cost = ?POSITIVE_INF, size = 5}, 
	Answer = task_handler(Task, #answer{jumps = [], cost = ?POSITIVE_INF}, 0, 0, false),
	io:format("[NEW CONNECTION HANDLER]Answer ~w~n", [Answer#answer.cost]).


new_connection_handler(ClientSocket, WorkingQueue) ->
	%Deserialize
	WorkingQueue ! {new_task, self()},
	Matrix = [
    [?POSITIVE_INF,            25,            40,            31,            27],
    [            5, ?POSITIVE_INF,            17,            30,            25],
    [           19,            15, ?POSITIVE_INF,             6,             1],
    [            9,            50,            24, ?POSITIVE_INF,             6],
    [           22,             8,             7,            10, ?POSITIVE_INF]],
    Mapping = lists:seq(0, 4),
    %find_heaviest_zero(Matrix).
    Task = #task{matrix = Matrix, row_mapping = Mapping, col_mapping = Mapping, jumps = [],
        curr_cost = 0, min_cost = ?POSITIVE_INF, size = 5}, 
	Answer = task_handler(Task, #answer{jumps = [], cost = ?POSITIVE_INF}, 0, 0, false),
	io:format("[NEW CONNECTION HANDLER]Answer ~w~n", [Answer#answer.cost]).

strtoint(Str)->
	[begin {Int,_}=string:to_integer(Token), Int end|| Token<-string:tokens(Str," ")].
val(X) ->
	[#xmlElement{name = N, content = [#xmlText{value = V}|_]}] = X,
	{N, V}.

serialize(Data) ->
	Xml = lists:flatten(xmerl:export_simple(Data, xmerl_xml)).

creatXml(Inpsize, Inpid, Inparray)->
  	RootElem = {qucksortCall, [{size, [Inpsize]}, {id, [Inpid]}, {array, [Inparray]}]},
  	serialize([RootElem]).

inttostr([])->
  	"";
inttostr([Pivot|Tail])->
  	if length(Tail)==0 -> string:concat(integer_to_list(Pivot), inttostr(Tail));
    	true -> string:concat(integer_to_list(Pivot), ","++inttostr(Tail))
  	end.
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
	WorkerPID = spawn(fun() -> worker(FreeWorkerQueuePID) end), 
	new_connection_handler([], TaskQueuePID).


























