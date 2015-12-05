import mpi4py.MPI as MPI
import enum
import asyncio

__author__ = 'Alex Uzhegov'


class Task:
    def __init__(self, data, callback, task_solver_functions_factory, *args, log):
        task_generator, intermediate_answer_reducer, final_answer = task_solver_functions_factory()
        self.subtasks = task_generator(data)
        self.intermediate_answer_reducer = intermediate_answer_reducer
        self.final_answer = final_answer
        self.callback = callback
        self.number_of_generated_subtasks = 0
        self.number_of_finished_subtasks = 0
        self.is_finished_generating_subtasks = False
        self.log = log

    def is_finished(self):
        self.log.info('[Task] {} {} {}'.format(self.is_finished_generating_subtasks, self.number_of_generated_subtasks, self.number_of_finished_subtasks))
        return self.is_finished_generating_subtasks and \
            self.number_of_generated_subtasks == self.number_of_finished_subtasks

    def __iter__(self):
        return self.sub_tasks_generator()

    def sub_tasks_generator(self):
        for subtask in self.subtasks:
            self.number_of_generated_subtasks += 1
            yield subtask
        self.is_finished_generating_subtasks = True
        raise StopIteration

    def process_answer_to_subtask(self, answer):
        self.log.info('[Task][process_answer_to_subtask] new sub answer to process ')
        self.intermediate_answer_reducer(answer)
        self.number_of_finished_subtasks += 1
        self.log.info('[Task][process_answer_to_subtask] new sub answer to ended. Ready {} out of {}.'.format(
            self.number_of_finished_subtasks, self.number_of_generated_subtasks
        ))

    def execute_callback(self):
        self.callback(self.final_answer())


class TaskSolverServer:
    class MPIMessageTag(enum.IntEnum):
        TaskRequest = 0
        TaskInfo    = 1
        TaskAnswer  = 2

    def __init__(self, comm, task_solver_functions_factory, *args, log):
        self.task_list = dict()
        self.task_queue = asyncio.Queue()
        self.free_workers = asyncio.Queue()
        self.answer_queue = asyncio.Queue()
        self.task_waiting_to_be_finished = asyncio.Queue()
        self.comm = comm
        self.task_factory = lambda task_data, callback: Task(task_data, callback, task_solver_functions_factory, log=self.log)
        self.task_id_generator = 0
        self.sleep_timeout = 0.01
        self.log = log

    def run(self):
        loop = asyncio.get_event_loop()
        tasks = [
            asyncio.ensure_future(self.free_worker_init()),
            asyncio.ensure_future(self.task_distributor()),
            asyncio.ensure_future(self.task_answer_handler()),
            asyncio.ensure_future(self.message_processor()),
            asyncio.ensure_future(self.task_completer_queue())]
        loop.run_until_complete(asyncio.wait(tasks))
        loop.close()

    def get_coroutines(self):
        return [
            asyncio.ensure_future(self.free_worker_init()),
            asyncio.ensure_future(self.task_distributor()),
            asyncio.ensure_future(self.task_answer_handler()),
            asyncio.ensure_future(self.message_processor()),
            asyncio.ensure_future(self.task_completer_queue())
        ]

    async def message_processor(self):
        status = MPI.Status()
        while True:
            while not self.comm.Iprobe(MPI.ANY_SOURCE, MPI.ANY_TAG, status):
                self.log.debug('[TaskSolverServer][message_processor] Waiting for messages...')
                await asyncio.sleep(self.sleep_timeout)
            source, tag = status.Get_source(), status.Get_tag()
            message_data = self.comm.recv(source=source, tag=tag)
            self.log.info('[TaskSolverServer][message_processor] New message from {} with tag = {} with data = {}'
                          .format(source, TaskSolverServer.MPIMessageTag(tag), message_data)
            )
            if tag == TaskSolverServer.MPIMessageTag.TaskAnswer:
                self.log.info('[TaskSolverServer][message_processor] Got new answer from {}'.format(source))
                await self.free_workers.put(source)
                await self.answer_queue.put(message_data)
            else:
                self.log.info('[TaskSolverServer][message_processor] Got wrong message')

    async def task_distributor(self):
        while True:
            task_id, subtask = -1, None
            while subtask is None:
                self.log.info('[TaskSolverServer][task_distributor] Waiting for new task...')
                task_id, task_generator = await self.task_queue.get()
                try:
                    subtask = next(task_generator)
                except StopIteration:
                    self.log.info('[TaskSolverServer][task_distributor] Sub task generator for task with id {} exhausted...'.format(task_id))
                    await self.task_waiting_to_be_finished.put(task_id)

            self.log.info('[TaskSolverServer][task_distributor] Got new task. Waiting for free worker...')
            free_worker = await self.free_workers.get()
            self.log.info('[TaskSolverServer][task_distributor] Got free worker {}.'.format(free_worker))
            self.log.info('[TaskSolverServer][task_distributor][DEBUG] {} {} {}'.format((task_id, subtask), free_worker, int(TaskSolverServer.MPIMessageTag.TaskInfo)))
            req = self.comm.isend((task_id, subtask), dest=free_worker, tag=int(TaskSolverServer.MPIMessageTag.TaskInfo))
            self.log.info('[TaskSolverServer][task_distributor] Sent new task with id {} to {}'.format(task_id, free_worker))
            while req.test() != (True, None):
                self.log.debug('[TaskSolverServer][task_distributor] Couldnt send to {}. Waiting...'.format(free_worker))
                await asyncio.sleep(self.sleep_timeout)
            self.log.info('[TaskSolverServer][task_distributor] Completed transmission of task to free worker'.format(task_id, free_worker))
            await self.task_queue.put((task_id, task_generator))
            self.log.info('[TaskSolverServer][task_distributor] Added task generator for task with id {} to task_queue...'.format(task_id))

    async def task_answer_handler(self):
        while True:
            self.log.info('[TaskSolverServer][task_answer_handler] Waiting for new sub answer')
            task_id, task_answer = await self.answer_queue.get()
            self.log.info('[TaskSolverServer][task_answer_handler] New sub answer came for task with id {}'.format(task_id))
            task = self.task_list[task_id]
            self.log.info('[TaskSolverServer][task_answer_handler] get task instance ')
            task.process_answer_to_subtask(task_answer)

    async def solve_new_task(self, data, callback):
        self.log.info('[TaskSolverServer][solve_new_task] New task arrived')
        task = self.task_factory(data, callback)
        self.log.info('[TaskSolverServer][solve_new_task] Created new task instance')
        task_id = self.get_new_task_id()
        self.log.info('[TaskSolverServer][solve_new_task] Generated new task id {}'.format(task_id))
        self.task_list[task_id] = task
        self.log.info('[TaskSolverServer][solve_new_task] Put new task into task_list')
        await self.task_queue.put((task_id, iter(task)))
        self.log.info('[TaskSolverServer][solve_new_task] Put sub task generator into queue')
        return task_id

    async def free_worker_init(self):
        for i in range(2, self.comm.Get_size()):
            await self.free_workers.put(i)

    async def task_completer_queue(self):
        i = 0
        while True:
            while not self.task_waiting_to_be_finished.empty():
                self.log.info('[TaskSolverServer][task_completer_queue] New unfinished task')
                task_id = await self.task_waiting_to_be_finished.get()
                self.log.info('[TaskSolverServer][task_completer_queue] got new id')
                task = self.task_list[task_id]
                self.log.info('[TaskSolverServer][task_completer_queue] got task instance')
                while not task.is_finished():
                    self.log.debug('[TaskSolverServer][task_completer_queue] Waiting for task to be finished...')
                    await asyncio.sleep(self.sleep_timeout)
                else:
                    self.log.info('[TaskSolverServer][task_completer_queue] Task finished. Getting rid of it')
                    del self.task_list[task_id]
                    task.execute_callback()
                    self.log.info('[TaskSolverServer][task_completer_queue] Completed task with id {}'.format(task_id))
                    break
            else:
                self.log.debug('[TaskSolverServer][task_completer_queue] Waiting for unfinished tasks {}'.format(i))
                await asyncio.sleep(self.sleep_timeout)

    def get_new_task_id(self):
        task_id = self.task_id_generator
        self.task_id_generator += 1
        return task_id


def protocol_factory(server, new_input_callback, output_ready_callback, log):
    class ClientHandlerServerProtocol(asyncio.Protocol):
        def connection_made(self, transport):
            peername = transport.get_extra_info('peername')
            log.info('Connection from {}'.format(peername))
            self.transport = transport

        def data_received(self, data):
            def answer_ready_callback(answer):
                output = output_ready_callback(answer)
                log.info('[data_received] Before write to socket')
                self.transport.write(output)
                log.info('[data_received] After write to socket')
                self.transport.close()
                log.info('[data_received] After socket close')

            asyncio.ensure_future(server.solve_new_task(new_input_callback(data), answer_ready_callback))
    return ClientHandlerServerProtocol


class Worker:
    def __init__(self, comm, task_handler, *args, log):
        self.conn = comm
        self.rank = comm.Get_rank()
        self.task_handler = task_handler
        self.sleep_timeout = 0.01
        self.log = log

    def run(self):
        status = MPI.Status()
        while True:
            import time
            # Receive task data
            while not self.conn.Iprobe(MPI.ANY_SOURCE, MPI.ANY_TAG, status):
                # print('[Worker({})]: Waiting for messages...'.format(self.rank))
                time.sleep(self.sleep_timeout)
            self.log.info('[Worker({})]: Receive task data'.format(self.rank))
            task_id, task = self.conn.recv(source=0, tag=int(TaskSolverServer.MPIMessageTag.TaskInfo))
            # Solve
            answer = self.task_handler(task)
            # Send task answer
            self.log.info('[Worker({})]: Send task answer {}'.format(self.rank, answer))
            self.conn.send((task_id, answer), dest=0, tag=int(TaskSolverServer.MPIMessageTag.TaskAnswer))


def run_server(*args, frontend_server_connection_info, frontend_server_callback_factory,
               task_solver_func_factory, worker_task_handler, log):
    if MPI.COMM_WORLD.Get_rank() == 0:
        server = TaskSolverServer(MPI.COMM_WORLD, task_solver_func_factory, log=log)
        loop = asyncio.get_event_loop()
        input_callback, output_callback = frontend_server_callback_factory()
        tasks = [asyncio.ensure_future(
            loop.create_server(
                protocol_factory(server, input_callback, output_callback, log),
                *frontend_server_connection_info
            )
        )]
        tasks.extend(server.get_coroutines())
        try:
            loop.run_until_complete(asyncio.wait(tasks))
        except KeyboardInterrupt:
            loop.close()
    else:
        worker = Worker(MPI.COMM_WORLD, worker_task_handler, log=log)
        try:
            worker.run()
        except KeyboardInterrupt:
            pass
