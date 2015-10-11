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

MainServer: module
{
	init: fn(nil: ref Draw->Context, argv: list of string);
};

Qroot, Qhi, Qexit : con big iota;


init_internet_connection(): (int, sys->Connection)
{
	sys->print("TRYING TO CONNECT\n");
	a := sys->dial("tcp!127.0.0.1!5000", "");
	sys->print("CONNECTED\n");

	return a;
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;

	a := init_internet_connection();
	
	if (a.t0 == -1)return;
	result := sys->mount(a.t1.dfd, nil, ".", sys->MBEFORE, nil);
	if (result == -1) return;

	sys->print("TRYING_TO_OPEN\n");
	fd := sys->open("./hi", sys->ORDWR);
	sys->print("OPENED\n");
	name_raw := array of byte "Bob";
	sys->print("TRYING_TO_WRITE\n");
	size := sys->write(fd, name_raw, len name_raw);
	sys->print("WRITTEN\n");
	if (size != len name_raw) return;

	output := array[100] of byte;
	sys->print("TRYING_TO_READ\n");
	read_size := sys->read(fd, output, 100);
	sys->print("READ\n");
	sys->print("%s", string output[0:read_size]);

	sys->print("READ0\n");
	exit_fd := sys->open("./exit", sys->ORDWR);
	if (exit_fd == nil) raise "wtf2";
	sys->print("READ1\n");
	size = sys->write(exit_fd, array[0] of byte, 0);
	sys->print("READ2\n");
	sys->unmount(nil, nil);
	sys->print("end\n");
	sys->write(a.t1.cfd, array of byte "hangup", len array of byte "hangup");
	return;
	sys->print("true end\n");
}