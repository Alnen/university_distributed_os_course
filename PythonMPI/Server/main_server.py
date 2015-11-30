import mpi4py.MPI as MPI
import time
from queue import Queue
from threading import Lock
import asyncio

__author__ = 'Alex Uzhegov'


global_task_deque = Queue()
answer_dictionary = dict()
dict_lock = Lock()


async def consumer():
    pass


def producer():
    pass

if __name__ == '__main__':
    comm = MPI.COMM_WORLD
    rank = comm.Get_rank()
    size = comm.Get_size()

    if rank == 0:
        # Server
        waiting_workers = list()
        count = 0
        async def getNextFreeWorker():
            nonlocal comm
            return comm.recv(source=MPI.ANY_SOURCE, tag=0)

        async def sendTaskToNextWorker(worker_id, task):
            nonlocal comm
            comm.send(task, dest=worker_id, tag=1)

        async def sendTaskToNextWorker(worker_id, task):
            nonlocal comm
            return comm.recv(source=MPI.ANY_SOURCE, tag=2)

        while True:
            # Gather task requests
            data = comm.recv(source=MPI.ANY_SOURCE, tag=0)
            waiting_workers.append(data)
            # Gather send task
            comm.send("HELLO {} out of {}, it's {} request".format(data, size, count), dest=data, tag=1)
            count += 1
            time.sleep(1)
            # Gather answers

    else:
        # Worker
        while True:
            # Send task request
            comm.send(rank, dest=0, tag=0)
            # Receive task data
            data = comm.recv(source=0, tag=1)
            # Solve
            print(data)
            # Send task answer

    print("Server")


