implement MainServer;

include "sys.m";
	sys: Sys;
include "draw.m";
include "styx.m";
	styx: Styx;
	Tmsg, Rmsg: import styx;
include "styxservers.m";
	styxservers: Styxservers;
	Styxserver, Navigator: import styxservers;
	nametree: Nametree;
	Tree: import nametree;
include "rand.m";
	rand: Rand;
include "string.m";
	str: String;
include "dts_solver.m";
	dts: dts_solver;

MainServer: module
{
	init: fn(nil: ref Draw->Context, argv: list of string);
};

Qroot, Qhi, Qexit : con big iota;

task_count := 0;


init_internet_connection(): (int, sys->Connection)
{
	while(1)
	{
		sys->print("TRYING TO CONNECT\n");
		a := sys->dial("tcp!127.0.0.1!5000", "");

		if (a.t0 != -1)
		{
			sys->print("CONNECTED\n");
			return a;
		} 
		else
		{
			sys->sleep(1000);
		}
	}
}

serialize_answer(answer : (int, ref dts->AnswerType)) : array of byte
{
	str_data := "";
	str_data += string answer.t0;
	str_data += " " + string answer.t1.t1;
	str_data += " " + string len answer.t1.t0;

	jumps := answer.t1.t0;
	while (jumps != nil)
	{
		curr_jump := hd jumps;
		jumps = tl jumps;

		str_data += " " + string curr_jump.t0;
		str_data += " " + string curr_jump.t1;
	}
	return array of byte str_data;
}

deserialize_task(str_data: string): (int, int, array of int, array of int, array of int, list of dts->JumpType, int, int)
{
	id := 0;
	size := 0;
	jumps : list of dts->JumpType;
	jumps = nil;
	curr_cost := 0;
	max_cost := dts->POSITIVE_INF;

	(id, str_data) = str->toint(str_data, 10);
	(size, str_data) = str->toint(str_data, 10);
	#
	matrix := array[size*size] of int;
	x_mapping := array[size] of int;
	y_mapping := array[size] of int;

	#
	for (i:=0;i<size*size;++i)
	{
		(matrix[i], str_data) = str->toint(str_data, 10);
	}

	for (i=0;i<size;++i)
	{
		(x_mapping[i], str_data) = str->toint(str_data, 10);
	}
	for (i=0;i<size;++i)
	{
		(y_mapping[i], str_data) = str->toint(str_data, 10);
	}

	jump_size := 0;
	(jump_size, str_data) = str->toint(str_data, 10);
	for (i=0;i<jump_size;++i)
	{
		from_, to_ : int;
		(from_, str_data) = str->toint(str_data, 10);
		(to_, str_data) = str->toint(str_data, 10);
		jumps = (from_, to_) :: jumps;
	}

	(curr_cost, str_data) = str->toint(str_data, 10);
	(max_cost, str_data) = str->toint(str_data, 10);

	return (id, size, matrix, x_mapping, y_mapping, jumps, curr_cost, max_cost);
}

worker(c: sys->Connection)
{
	result := sys->mount(c.dfd, nil, ".", sys->MBEFORE, nil);
	if (result == -1) return;

	sys->print("TRYING_TO_OPEN\n");
	fd := sys->open("./hi", sys->ORDWR);
	if (fd == nil) raise "WTF";
	sys->print("OPENED\n");
	while(1)
	{

		#read task
		output := array[10000] of byte;
		sys->print("TRYING TO READ NEW TASK\n");
		read_size := sys->read(fd, output, 10000);
		sys->print("DONE READING\n");
		real_data := string output[0:read_size];

		data_string := string real_data;
		task := deserialize_task(data_string);


		#do it
		sys->print("TRYING TO SOLVE TASK cost: %d min_const: %d size: %d\n", task.t6, task.t7, task.t1);
		answer := dts->solve_impl(task.t2, task.t3, task.t4, task.t5, task.t6, task.t7, task.t1);
		sys->print("SOLVED cost: %d min_cost: %d\n", answer.t1, task.t7);

		#send answer

		out_data := serialize_answer((task.t0, ref answer));
		#data := "";
		#data = string size + data;
		#raw_data := array of byte data;
		#raw_size := len raw_data;

		sys->print("TRYING TO SEND TASK № %d\n", task_count++);
		size := sys->write(fd, out_data, len out_data);
		if (size != len out_data) raise "WTF";
		sys->print("SENT\n");
	}
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	str = load String String->PATH;
	dts = load dts_solver "dts_solver.dis";

	a := init_internet_connection();
	
	while(1)
	{
		{
			worker(a.t1);
		}
		exception e
		{
			"*" => 
				a = init_internet_connection();
				task_count = 0;
		}
	}
	
	sys->unmount(nil, nil);
	sys->write(a.t1.cfd, array of byte "hangup", len array of byte "hangup");
	return;
}