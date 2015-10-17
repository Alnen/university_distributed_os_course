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
include "dial.m";
	dial: Dial;
include "string.m";
	str: String;
include "dts_solver.m";
	dts: dts_solver;
include "Lists.m";
	lists: Lists;

MainServer: module
{
	init: fn(nil: ref Draw->Context, argv: list of string);
};

Qroot, Qhi, Qexit : con big iota;

tree : ref Nametree->Tree;
treeop : chan of ref Styxservers->Navop;
nav : ref Styxservers->Navigator;

self_con : sys->Connection;
_ : int;


##
consumer(channel: chan of ref dts->AnswerType, number_of_tasks: int, final_answer_cchannel: chan of ref dts->AnswerType, min_value: ref atomic_int, num: chan of int)
{
	sys->print("[CLIENT] : CONSUMER START\n");
	best_solution : ref dts->AnswerType;
	best_solution = ref (nil, dts->POSITIVE_INF);
	count := 0;
	for(i := 0;<-num; ++i)
	{

		possible_solution := <- channel;
		sys->print("[CLIENT] : recieved new task %d from %d of %d\n", count++, i, number_of_tasks);
		if (possible_solution.t1 != dts->POSITIVE_INF && possible_solution.t1 < best_solution.t1)
		{
			min_value.set(possible_solution.t1);
			best_solution = possible_solution;
			sys->print("[CLIENT] : current min_value = %d\n", min_value.get());
		}
	}
	sys->print("[CLIENT] : ready to send final data\n");
	final_answer_cchannel<- = best_solution;
}

##
atomic_int: adt
{
	value: int;
	mutex: chan of int;

	new: fn(val: int): ref atomic_int;
	get: fn(a: self ref atomic_int) : int;
	set: fn(a: self ref atomic_int, val: int);
};

atomic_int.new(val: int): ref atomic_int
{
	a : atomic_int;
	a.value = val;
	a.mutex = chan[1] of int;
	a.mutex <- =  1;

	return ref a;
}

atomic_int.get(a: self ref atomic_int) : int
{
	<-a.mutex;
	v := a.value;
	a.mutex<- = 1;
	return v;
}
atomic_int.set(a: self ref atomic_int, val: int)
{
	<-a.mutex;
	a.value = val;
	a.mutex<- = 1;
}


##
task_generator()
{
	sys->print("[CLIENT] : STARTING\n");
	buffer := array[100] of byte;
	#size := sys->read(sys->fildes(0), buffer, 100);
	#real_data := string buffer[0:size];
	task_size : int;
	#(task_size, real_data) = str->toint(real_data, 10);
	sys->print("[CLIENT] : STARTING\n");
	task_size = 30;
	task := dts->generate_random_task_of_size_n(task_size, 100, 0);
	sys->print("[CLIENT] : STARTING\n");

	stack : list of ref dts->TaskType;
	tmp : dts->TaskType;
	tmp = (task.t0, task.t1, task.t2, nil, 0, dts->POSITIVE_INF, task.t3);
	stack = ref tmp :: stack;

	flag := 0;
	sub_task1, sub_task2 : dts->TaskType;
	sys->print("[CLIENT] : *\n");
	answer_adt := add_output_chan_list();
	number_of_tasks := 0;
	sys->print("[CLIENT] : INITIALIZED\n");

	#split
	do
	{
		current_tasl := hd stack;
		stack = tl stack;
		(flag, (sub_task1, sub_task2)) = dts->crusher_impl(current_tasl.t0, current_tasl.t1, current_tasl.t2, current_tasl.t3, current_tasl.t4, current_tasl.t5, current_tasl.t6);
		sys->print("[CLIENT] : flag: %d \n", flag);
		if (flag)
		{
			stack = ref sub_task2 :: ref sub_task1 :: stack;
		}
	}
	while(flag);
	stack = lists->reverse(stack);
	sys->print("[CLIENT] : SOLVING\n");

	#solve
	min_value := atomic_int.new(dts->POSITIVE_INF);
	num := chan[10] of int;

	number_of_tasks = len stack;
	final_answer_cchannel := chan[1] of ref dts->AnswerType;
	spawn consumer(answer_adt.channel, number_of_tasks, final_answer_cchannel, min_value, num);

	count := 0;

	while (stack != nil)
	{
		current_task := hd stack;
		stack = tl stack;
		min_val := min_value.get();
		sys->print("[CLIENT] : sent new task id:%d count:%d cost:%d min_const: %d| of %d\n", count++, answer_adt.id, current_task.t4, min_val, number_of_tasks);
		if (current_task.t4 > min_val) continue;

		input_global_task_queue<- = (answer_adt.id, current_task.t6, current_task.t0, current_task.t1, current_task.t2, current_task.t3, current_task.t4, min_val);
		num<- = 1;
	}
	num<- = 0;
	#collect
	sys->print("[CLIENT] : COLLECT\n");
	best_solution := <- final_answer_cchannel;
	#print answer 
	if (best_solution.t1 != dts->POSITIVE_INF)
	{
		sys->print("***TASK SOLVED\n");
	}
	else
	{
		sys->print("***TASK NOT SOLVE\n");
	}
	delete_output_chan_list(answer_adt);
}

##
self_connect()
{
	(_, self_con) = sys->dial("tcp!127.0.0.1!5000", "");
	result := sys->mount(self_con.dfd, nil, ".", sys->MBEFORE, nil);
	if (result == -1) raise "aba";
}

self_disconnect()
{
	sys->unmount(nil, nil);
	sys->write(self_con.cfd, array of byte "hangup", len array of byte "hangup");
}

run_styx_server(c : ref dial->Connection)
{
	(tchan, srv) := Styxserver.new(c.dfd, nav, Qroot);
	spawn server(tchan, srv, tree);	
}

server_thread(c : ref dial->Connection)
{
	while(1)
	{
		sys->print("WAITING FOR CONNECTION\n");
		a := dial->listen(c);
		sys->print("NEW CONNECTION\n");
		fd := dial->accept(a);
		a.dfd = fd;
		sys->print("ACCEPTED IT\n");

		spawn run_styx_server(a);
	}
}

run_server()
{
	c := dial->announce("tcp!127.0.0.1!5000");
	sys->print("ANNOUNCED\n");
	if (c != nil)
	{
		server_thread(c);
	}
	return ;
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	styx = load Styx Styx->PATH;
	dial = load Dial Dial->PATH;
	dts = load dts_solver "dts_solver.dis";
	str = load String String->PATH;
	lists = load Lists Lists->PATH;

	dts->init1();

	sys->print("1\n");

	output_chan_mutex = chan[1] of int;
	output_chan_mutex <- = 1;
	output_chan_list = nil;

	styx->init();
	styxservers = load Styxservers Styxservers->PATH;
	styxservers->init(styx);
	nametree = load Nametree Nametree->PATH;
	nametree->init();

	(tree, treeop) = nametree->start();

	tree.create(Qroot, dir(".", 8r555|Sys->DMDIR, Qroot));
	tree.create(Qroot, dir("hi", 8r666, Qhi));
	tree.create(Qroot, dir("exit", 8r666, Qexit));

	nav = Navigator.new(treeop);

	input_global_task_queue = chan[100] of (int, int, array of int, array of int, array of int, list of dts->JumpType, int, int);
	
	sys->print("HI\n");
	spawn task_generator();
}

dir(name: string, perm: int, qid: big): Sys->Dir
{
	d := sys->zerodir;
	d.name = name;
	d.uid = "me";
	d.gid = "me";
	d.qid.path = qid;
	if (perm & Sys->DMDIR)	
		d.qid.qtype = Sys->QTDIR;
	else				
		d.qid.qtype = Sys->QTFILE;
	d.mode = perm;
	return d;
}

deserialize_vector(s: string, size: int): (array of int, string)
{
	val := 0;
	vec := array[size] of int;

	for(i := 0; i < size; ++i)
	{
		(val, s) = str->toint(s, 10);
		vec[i] = val;
	}

	return (vec, s);
}

deserialize(s: string): (int, array of int, array of int)
{
	size := 0;
	(size, s) = str->toint(s, 10);

	vec1, vec2 : array of int;

	(vec1, s) = deserialize_vector(s, size);
	(vec2, s)  = deserialize_vector(s, size);
	return (size, vec1, vec2);
}

serialize_vector(vec: array of int, size: int): string
{
	data := "";	
	for (i := 0; i<size ; ++i)
	{
		data += " " + string vec[i];
	}
	return data;
}

serialize(vec: array of int, size: int): string
{
		data := "";
		data += string size;
		data += serialize_vector(vec, size);
		return data;
}

input_global_task_queue : chan of (int, int, array of int, array of int, array of int, list of dts->JumpType, int, int);

output_chan_list : list of ref AnswerTypeADT;
output_chan_mutex : chan of int;

id_gen := int 10;
lock()
{
	lock := <-output_chan_mutex;
}

unlock()
{
	output_chan_mutex<- = 1;
}

AnswerTypeADT : adt
{
	id : int;
	channel : chan of ref dts->AnswerType;

	eq: fn(lhs: ref AnswerTypeADT, rhs : ref AnswerTypeADT): int;
};

AnswerTypeADT.eq(lhs: ref AnswerTypeADT, rhs : ref AnswerTypeADT): int
{
	return lhs.id == rhs.id;
}

add_output_chan_list(): ref AnswerTypeADT
{
	lock();

	new_id := id_gen++;
	channel := chan[10] of ref dts->AnswerType;
	#
	val : AnswerTypeADT;
	val.id = new_id;
	val.channel = channel;
	#
	output_chan_list = lists->append(output_chan_list, ref val);

	# add to tree
	#tree.create(Qroot, dir("out_" + string val.id, 8r666, big val.id));
	unlock();

	return ref val;
}

delete_output_chan_list(val : ref AnswerTypeADT)
{
	lock();

	lists->delete(val, output_chan_list);

	#delete from tree
	#tree.remove(big val.id);

	unlock();
}

deserialize_answer(data : array of byte): (int, ref dts->AnswerType)
{
	id : int;
	answer : dts->AnswerType;

	answer.t0 = nil;
	answer.t1 = 0;
	

	str_data := string data;
	(id, str_data) = str->toint(str_data, 10);
	(answer.t1, str_data) = str->toint(str_data, 10);

	jumps_count := 0;
	(jumps_count, str_data) = str->toint(str_data, 10);
	for (i:=0; i<jumps_count;++i)
	{
		temp1,temp2 : int;
		(temp1, str_data) = str->toint(str_data, 10);
		(temp2, str_data) = str->toint(str_data, 10);
		answer.t0 = (temp1, temp2) :: answer.t0;
	}
	return (id, ref answer);
}

serialize_task(data : (int, int, array of int, array of int, array of int, list of dts->JumpType, int, int)) : array of byte
{
	id := data.t0;
	size := data.t1;
	matrix := data.t2;
	x_mapping := data.t3;
	y_mapping := data.t4;
	jumps := data.t5;
	curr_cost := data.t6;
	max_cost := data.t7;

	str_data := "";
	str_data += " " + string id;
	str_data += " " + string size;
	#
	for (i:=0;i<size*size;++i)
	{
		str_data += " " + string matrix[i];
	}

	for (i=0;i<size;++i)
	{
		str_data += " " + string x_mapping[i];
	}
	for (i=0;i<size;++i)
	{
		str_data += " " + string y_mapping[i];
	}

	jump_size := len jumps;
	str_data += " " + string jump_size;
	for (i=0;i<jump_size;++i)
	{
		head := hd jumps;
		jumps = tl jumps;
		str_data += " " + string head.t0;
		str_data += " " + string head.t1;
	}
	#
	str_data += " " + string curr_cost;
	str_data += " " + string max_cost;

	return array of byte str_data;
}


server(tchan: chan of ref Tmsg, srv: ref Styxserver, tree: ref Tree)
{
    sys->pctl(Sys->NEWPGRP, nil);

    closed := 0;
    
    while(!closed && ((gm := <-tchan) != nil)) 
    {

		#sys->print("[SERVER] : NEW MESSAGE\n");
	    pick m := gm 
	    {
	    	Write =>
	    		sys->print("[SERVER] : NEW WRITE MESSAGE\n");
				(c, err) := srv.canwrite(m);
			
				if (c == nil)
				{
					srv.reply(ref Rmsg.Error(m.tag, err));
				} 
				else if (c.path == Qhi)
				{	
					sys->print("NEW TASK REQUEST\n");
					# deserialize task done
					(id, answer) := deserialize_answer(m.data);

					#find right cannal
					search_val : AnswerTypeADT;
					search_val.id = id;
					ans_adt := hd lists->find(ref search_val, output_chan_list);
					#push data
					ans_adt.channel <- = answer;

					srv.reply(ref Rmsg.Write(m.tag, len m.data));
					sys->print("SENT NEW TASK\n");
				}
				else if (c.path == Qexit){
					closed = 1;
					sys->print("WRITE TO Qexit\n");
					srv.reply(ref Rmsg.Write(m.tag, len m.data));
				}
		
			Read =>
				sys->print("[SERVER] : NEW READ MESSAGE\n");
				(c, err) := srv.canread(m);
				if (c == nil)
					srv.reply(ref Rmsg.Error(m.tag, err));
				sys->print("[SERVER] : *\n");
				case c.path {
					Qhi =>
						sys->print("[SERVER] : **\n");
						# serialize task
						task := <- input_global_task_queue;
						raw_task := serialize_task(task);
						srv.reply(ref Rmsg.Read(m.tag, raw_task));
						#styxservers->readbytes(m, data));
						
					* =>
						srv.default(gm);
				}
			* =>
				srv.default(gm);
		}
	}
	#tree.quit();
	#sys->print("at end\n");
}