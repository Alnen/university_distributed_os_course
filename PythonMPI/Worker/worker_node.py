import mpi4py.MPI as MPI
import enum
import asyncio
import socket
import random
import time

__author__ = 'Alex Uzhegov'


class Task:
    def __init__(self, data, callback, task_solver_functions_factory):
        task_generator, intermediate_answer_reducer, final_answer = task_solver_functions_factory()
        self.subtasks = task_generator(data)
        self.intermediate_answer_reducer = intermediate_answer_reducer
        self.final_answer = final_answer
        self.callback = callback
        self.number_of_generated_subtasks = 0
        self.number_of_finished_subtasks = 0
        self.is_finished_generating_subtasks = False

    def is_finished(self):
        print('[Task] ', self.is_finished_generating_subtasks, self.number_of_generated_subtasks, self.number_of_finished_subtasks)
        return self.is_finished_generating_subtasks and \
                self.number_of_generated_subtasks == self.number_of_finished_subtasks

    def __iter__(self):
        return self.sub_tasks_generator()

    def sub_tasks_generator(self):
        print('[*]0')
        for subtask in self.subtasks:
            print('[*]2')
            self.number_of_generated_subtasks += 1
            print('[*]3')
            yield subtask
            print('[*]4')
        print('[*]5')
        self.is_finished_generating_subtasks = True
        raise StopIteration

    def process_answer_to_subtask(self, answer):
        print('[Task][process_answer_to_subtask] new sub answer to process ', answer)
        self.intermediate_answer_reducer(answer)
        self.number_of_finished_subtasks += 1
        print('[Task][process_answer_to_subtask] new sub answer to ended')

    async def execute_callback(self):
        await self.callback(self.final_answer())


class TaskSolverServer:
    class MPIMessageTag(enum.IntEnum, ):
        TaskRequest = 0
        TaskInfo    = 1
        TaskAnswer  = 2

    def __init__(self, comm, task_solver_functions_factory):
        self.task_list = dict()
        self.task_queue = asyncio.Queue()
        self.free_workers = asyncio.Queue()
        self.answer_queue = asyncio.Queue()
        self.task_waiting_to_be_finished = asyncio.Queue()
        self.comm = comm
        self.task_factory = lambda task_data, callback: Task(task_data, callback, task_solver_functions_factory)
        self.task_id_generator = 0
        self.sleep_timeout = 0.01

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
                # print('[TaskSolverServer][message_processor] Waiting for messages...')
                await asyncio.sleep(self.sleep_timeout)
            source, tag = status.Get_source(), status.Get_tag()
            message_data = self.comm.recv(source=source, tag=tag)
            print('[TaskSolverServer][message_processor] New message from {} with tag = {} with data = {}'.format(source, TaskSolverServer.MPIMessageTag(tag), message_data))
            if tag == TaskSolverServer.MPIMessageTag.TaskAnswer:
                print('[TaskSolverServer][message_processor] Got new answer from', source)
                await self.free_workers.put(source)
                await self.answer_queue.put(message_data)
            else:
                print('[TaskSolverServer][message_processor] Got wrong message')

    async def task_distributor(self):
        while True:
            task_id, subtask = -1, None
            while subtask is None:
                print('[TaskSolverServer][task_distributor] Waiting for new task...')
                task_id, task_generator = await self.task_queue.get()
                try:
                    subtask = next(task_generator)
                except StopIteration:
                    print('[TaskSolverServer][task_distributor] Sub task generator for task with id {} exhausted...'.format(task_id))
                    await self.task_waiting_to_be_finished.put(task_id)

            print('[TaskSolverServer][task_distributor] Got new task. Waiting for free worker...')
            free_worker = await self.free_workers.get()
            print('[TaskSolverServer][task_distributor] Got free worker {}.'.format(free_worker))
            print('[TaskSolverServer][task_distributor][DEBUG] {} {} {}'.format((task_id, subtask), free_worker, int(TaskSolverServer.MPIMessageTag.TaskInfo)))
            req = self.comm.isend((task_id, subtask), dest=free_worker, tag=int(TaskSolverServer.MPIMessageTag.TaskInfo))
            print('[TaskSolverServer][task_distributor] Sent new task with id {} to {}'.format(task_id, free_worker))
            while req.test() != (True, None):
                # print('[TaskSolverServer][task_distributor] Couldnt send to {}. Waiting...'.format(free_worker))
                await asyncio.sleep(self.sleep_timeout)
            print('[TaskSolverServer][task_distributor] Completed transmission of task to free worker'.format(task_id, free_worker))
            await self.task_queue.put((task_id, task_generator))
            print('[TaskSolverServer][task_distributor] Added task generator for task with id {} to task_queue...'.format(task_id))

    async def task_answer_handler(self):
        while True:
            print('[TaskSolverServer][task_answer_handler] Waiting for new sub answer')
            task_id, task_answer = await self.answer_queue.get()
            print('[TaskSolverServer][task_answer_handler] New sub answer came for task with id {}'.format(task_id))
            task = self.task_list[task_id]
            print('[TaskSolverServer][task_answer_handler] get task instance ')
            task.process_answer_to_subtask(task_answer)

    async def solve_new_task(self, data, callback):
        print('[TaskSolverServer][solve_new_task] New task arrived')
        task = self.task_factory(data, callback)
        print('[TaskSolverServer][solve_new_task] Created new task instance')
        task_id = self.get_new_task_id()
        print('[TaskSolverServer][solve_new_task] Generated new task id {}'.format(task_id))
        self.task_list[task_id] = task
        print('[TaskSolverServer][solve_new_task] Put new task into task_list')
        await self.task_queue.put((task_id, iter(task)))
        print('[TaskSolverServer][solve_new_task] Put sub task generator into queue')
        return task_id

    async def free_worker_init(self):
        for i in range(2, self.comm.Get_size()):
            await self.free_workers.put(i)

    async def task_completer_queue(self):
        while True:
            while not self.task_waiting_to_be_finished.empty():
                print('[TaskSolverServer][task_completer_queue] New unfinished task')
                task_id = await self.task_waiting_to_be_finished.get()
                print('[TaskSolverServer][task_completer_queue] got new id')
                task = self.task_list[task_id]
                print('[TaskSolverServer][task_completer_queue] got task instance')
                while not task.is_finished():
                    # print('[TaskSolverServer][task_completer_queue] Waiting for task to be finished...')
                    await asyncio.sleep(self.sleep_timeout)
                else:
                    print('[TaskSolverServer][task_completer_queue] Task finished. Getting rid of it')
                    del self.task_list[task_id]
                    await task.execute_callback()
                    print('[TaskSolverServer][task_completer_queue] Completed task with id {}'.format(task_id))
                    break
            else:
                # print('[TaskSolverServer][task_completer_queue] Waiting for unfinished tasks')
                await asyncio.sleep(self.sleep_timeout)

    def get_new_task_id(self):
        task_id = self.task_id_generator
        self.task_id_generator += 1
        return task_id


class EchoServerClientProtocol(asyncio.Protocol):
    def connection_made(self, transport):
        # asyncio.ensure_future(server.solve_new_task(list(range(100)), lambda x: print('[*****]Average is ', x)))
        peername = transport.get_extra_info('peername')
        print('Connection from {}'.format(peername))
        self.transport = transport

    def data_received(self, data):
        message = data.decode()
        print('Data received: {!r}'.format(message))

        loop = asyncio.get_event_loop()

        async def callback(asnwer):
            nonlocal self
            print('[callbacks]', asnwer)
            self.transport.write('{}'.format(asnwer).encode())
            self.close()
        fut = asyncio.ensure_future(server.solve_new_task([int(x) for x in message.split()], callback))
        fut.result()
        """print('Send: {!r}'.format(message))
        self.transport.write(data)

        print('Close the client socket')
        self.transport.close()"""


class Worker:
    def __init__(self, comm, task_handler):
        self.conn = comm
        self.rank = comm.Get_rank()
        self.task_handler = task_handler
        self.sleep_timeout = 0.01

    def run(self):
        status = MPI.Status()
        while True:
            import time
            # Receive task data
            while not self.conn.Iprobe(MPI.ANY_SOURCE, MPI.ANY_TAG, status):
                # print('[Worker({})]: Waiting for messages...'.format(self.rank))
                time.sleep(self.sleep_timeout)
            print('[Worker({})]: Receive task data'.format(self.rank))
            task_id, task = self.conn.recv(source=0, tag=int(TaskSolverServer.MPIMessageTag.TaskInfo))
            # Solve
            answer = self.task_handler(task)
            # Send task answer
            print('[Worker({})]: Send task answer {}'.format(self.rank, answer))
            self.conn.send((task_id, answer), dest=0, tag=int(TaskSolverServer.MPIMessageTag.TaskAnswer))


def generate_task(size):
    return " ".join((str(random.randint(0, 100)) for x in range(size)))


class Client:
    def __init__(self):
        self.server_ip = "127.0.0.1"
        self.server_port = 8888

    def run(self):
        task = generate_task(10000)
        task_binary = task.encode()
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        print("[CLIENT] Trying to connect to server at {}:{} ...".format(self.server_ip, self.server_port))
        flag = False
        while not flag:
            try:
                s.connect((self.server_ip, self.server_port))
                flag = True
            except ConnectionRefusedError:
                time.sleep(0.1)

        print("[CLIENT] Connecting done. Trying to send task...".format(self.server_ip, self.server_port))
        t1 = time.clock()
        s.send(task_binary)
        print("[CLIENT] Sending task done. Trying to receive answer...".format(self.server_ip, self.server_port))
        resp = s.recv(4096)
        print("[CLIENT] Received answer = ", resp.decode())
        s.close()
        return resp


if __name__ == '__main__':
    if MPI.COMM_WORLD.Get_rank() == 0:
        def average_task_solver_functions_factory():
            current_sum = 0
            size = 0

            def generate_new_task(task_data):
                nonlocal size
                for item in task_data:
                    size += 1
                    yield item

            def answer_reducer(task_intermediate_answer):
                nonlocal current_sum
                current_sum += task_intermediate_answer
                print('[TaskSolverServer] New value of task is {}'.format(current_sum))

            def final_answer():
                return current_sum / size

            return generate_new_task, answer_reducer, final_answer

        server = TaskSolverServer(MPI.COMM_WORLD, average_task_solver_functions_factory)
        loop = asyncio.get_event_loop()
        tasks = [asyncio.ensure_future(loop.create_server(EchoServerClientProtocol, '127.0.0.1', 8888))]
        tasks.extend(server.get_coroutines())
        loop.run_until_complete(asyncio.wait(tasks))
        loop.close()

    elif MPI.COMM_WORLD.Get_rank() == 1:
        client = Client()
        client.run()
    else:
        worker = Worker(MPI.COMM_WORLD, lambda x: 2*x)
        worker.run()

