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

MainServer: module
{
	init: fn(nil: ref Draw->Context, argv: list of string);
};

Qroot, Qhi, Qexit : con big iota;


init_internet_connection(): ref dial->Connection
{
	c := dial->announce("tcp!127.0.0.1!5000");
	sys->print("ANNOUNCED\n");
	if (c != nil)
	{
		sys->print("WAITING FOR CONNECTION\n");
		a := dial->listen(c);
		sys->print("NEW CONNECTION\n");
		fd := dial->accept(a);
		a.dfd = fd;
		sys->print("ACCEPTED IT\n");
		return a;
	}
	return c;
}

init(nil: ref Draw->Context, args: list of string)
{

	sys = load Sys Sys->PATH;
	styx = load Styx Styx->PATH;
	dial = load Dial Dial->PATH;
	styx->init();
	styxservers = load Styxservers Styxservers->PATH;
	styxservers->init(styx);
	nametree = load Nametree Nametree->PATH;
	nametree->init();

	(tree, treeop) := nametree->start();

	tree.create(Qroot, dir(".", 8r555|Sys->DMDIR, Qroot));
	tree.create(Qroot, dir("hi", 8r666, Qhi));
	tree.create(Qroot, dir("exit", 8r666, Qexit));

	nav := Navigator.new(treeop);

	a := init_internet_connection();
	sys->print("WAITING TO CREATE SERVER\n");
	if (a.dfd == nil) raise "wtf";
	(tchan, srv) := Styxserver.new(a.dfd, nav, Qroot);
	sys->print("CREATED SERVER\n");
	exit_condition := chan of int;
	spawn server(tchan, srv, tree, exit_condition);		
	(<- exit_condition);				
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

server(tchan: chan of ref Tmsg, srv: ref Styxserver, tree: ref Tree, exit_condition: chan of int)
{
    sys->pctl(Sys->NEWPGRP, nil);

    closed := 0;
    name := string "";
    
    while(!closed && ((gm := <-tchan) != nil)) 
    {
	    pick m := gm 
	    {
	    	Write =>
				(c, err) := srv.canwrite(m);
			
				if (c == nil)
				{
					srv.reply(ref Rmsg.Error(m.tag, err));
				} 
				else if (c.path == Qhi) 
				{
					name = string m.data;
					sys->print("WRITE TO Qhi : %s\n", name);
					srv.reply(ref Rmsg.Write(m.tag, len m.data));
				}
				else if (c.path == Qexit)
				{
					closed = 1;
					sys->print("WRITE TO Qexit\n");
					srv.reply(ref Rmsg.Write(m.tag, len m.data));
				}
		
			Read =>
				(c, err) := srv.canread(m);
				if (c == nil)
					srv.reply(ref Rmsg.Error(m.tag, err));

				case c.path {
					Qhi =>
						sys->print("READ FROM Qhi: %s\n", "Hello, you're in inferno! " + name);
						data := array of byte ("Hello, you're in inferno! " + name + "|abuday\n");  
						sys->print("%s\n", string data);
						srv.reply(ref Rmsg.Read(m.tag, data)); 
						#styxservers->readbytes(m, data));
						
					* =>
						srv.default(gm);
				}
			* =>
				srv.default(gm);
		}
	}
	tree.quit();
	sys->print("at end\n");
	exit_condition<- = 0;
}