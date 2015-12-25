-module(client_server).
-include("tsp_types.hrl").
-compile(solver).
-export([server/2,client/2,start/0,producer/1]).

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
	connection_listener_loop(ListenSocket).
%-------------------- New connection handeler ---------------------------
new_connection_handler(ClientSocket, WorkingQueue) ->
	%Deserialize
	WorkingQueue ! {new_task, self()}
	Answer = task_handler(Task, #answer{jumps = [], cost = ?POSITIVE_INF}, 0, 0, false),
	%Serialize

	% Send answer
	% Close socket

task_handler(Task, MinValue, NumberOfGeneratedTasks, NumberOfSolvedTasks, FinishedGenerationgTasks) ->
	receive ->
		{new_task_request, TaskQueuePID} ->
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
			{Success, SubTask, RestTasks} solver:crusher_impl(UpdatedTask),
			% Send task to queueu
			if
				Success =:= true ->
					TaskQueuePID ! {ok, SubTask},
					task_handler(RestTasks, MinValue, NumberOfGeneratedTasks+1, NumberOfSolvedTasks, FinishedGenerationgTasks);
				true ->
					TaskQueuePID ! {failed},
					task_handler(RestTasks, MinValue, NumberOfGeneratedTasks, NumberOfSolvedTasks, true)
			end;
		{handle_new_answer, NewAnswer} ->
			if
				MinValue#answer.cost > NewAnswer#answer.cost ->
            		NewMinValue = NewAnswer;
        		true -> 
        			NewMinValue = MinValue
    		end,
			if 
				FinishedGenerationgTasks =:= true && NumberOfSolvedTasks == NumberOfGeneratedTasks ->
					NewMinValue;
				true ->
					task_handler(Task, NewAnswer, NumberOfGeneratedTasks, NumberOfSolvedTasks+1, FinishedGenerationgTasks)		
			end
	end.
%-------------------- Worker Queue ---------------------------
worker_queue(State, FreeWorkerQueue) ->
	if 
		State == waiting_for_generator ->
			receive ->
				{new_task, TaskGeneratorPID} ->
					TaskGeneratorPID ! {new_task_request, self()}
					receive -> task
		State == sending_request
			receive ->
				{ok} ->
					FreeWorkerQueue ! {request, self()}
					receive ->
						{free_worker, WorkerPID} ->
					end;
				{failed} ->
					worker_queue(waiting_for_generator, FreeWorkerQueue)
			end
	end.


%-------------------- New connection handeler ---------------------------

%-------------------- New connection handeler ---------------------------

server(sending_task, Return_PID) ->
	receive 
		{append_task, Task} ->
			Return_PID ! {new_task, Task}
		after 1000 -> 
			io:format("[TASK_QUEUE~w] No task to send to ~w~n", [self(), Return_PID]),
			server(sending_task, Return_PID)
	end;

server(waiting_for_request, Param) -> 
	receive
		{request_task, Return_PID} ->
			io:format("[TASK_QUEUE~w] Worker requested task ~w~n",
						[self(), Return_PID]), 
			server(sending_task, Return_PID)
    end,
    server(waiting_for_request, []).
%-------------------- WORKER ---------------------------
client(waiting, Param) ->  
	receive
		{new_task, Task} ->
			io:format("[CLIENT~w] New task is ~w~n",[self(), Task])
		after 2000-> 
			io:format("[CLIENT~w] Still waiting for task~n",[self()])
	end,
	client(waiting, []);

client(sending, Task_Queue_Address) -> 
	io:format("[CLIENT~w] Sent request for new task to ~w~n",[self(), Task_Queue_Address]),
	Task_Queue_Address ! {request_task, self()},
	client(waiting, []),
	client(sending, Task_Queue_Address).
%-------------------- TASK_QUEUE ---------------------------
producer(Task_Queue_Address) -> 
	Task_Queue_Address ! {append_task, 10}.
%-------------------- ELSE ---------------------------

start() ->
	Server_PID = spawn(client_server,server,[waiting_for_request, []]), 
	spawn(client_server,client,[sending, Server_PID]).


spawn_n(Module, Function, ArgumentList, 0) -> ok;

spawn_n(Module, Function, ArgumentList, N) ->
	spawn(Module,Function,ArgumentList),
	io:format("[SPAWN_N] N = ~w~n", [N]),
	spawn_n(Module, Function, ArgumentList, N-1).

%start2() ->
%	Server_PID = spawn(client_server,server,[0]), 
%	spawn_n(client_server,client,[Server_PID], 5).
