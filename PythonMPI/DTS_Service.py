import logging
import sys
import Server.Server as Server

__author__ = 'Alex Uzhegov'


def generate_task(size):
    return " ".join((str(random.randint(0, 100)) for x in range(size)))


class Client:
    def __init__(self, *args, log):
        self.server_ip = "127.0.0.1"
        self.server_port = 8888
        self.log = log

    def run(self):
        task = generate_task(1000)
        task_binary = task.encode()
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.log.info("[CLIENT] Trying to connect to server at {}:{} ...".format(self.server_ip, self.server_port))
        flag = False
        while not flag:
            try:
                s.connect((self.server_ip, self.server_port))
                flag = True
            except ConnectionRefusedError:
                time.sleep(0.1)
        self.log.info("[CLIENT] Connecting done. Trying to send task...")
        t1 = time.clock()
        s.send(task_binary)
        self.log.info("[CLIENT] Sending task done. Trying to receive answer...")
        print("*** ***")
        resp = s.recv(4096)
        t2 = time.clock()
        print("*** {} ***".format(resp.decode()))
        self.log.info("[CLIENT] Received answer = {}. It took {} milliseconds.".format(resp.decode(), t2-t1))
        s.close()
        return resp


class MockLogger:
    def debug(msg, *args, **kwargs): pass

    def info(msg, *args, **kwargs): pass

    def warn(msg, *args, **kwargs): pass

    def error(msg, *args, **kwargs): pass

    def critical(msg, *args, **kwargs): pass

if __name__ == '__main__':
    log_handler = logging.StreamHandler(sys.stdout)
    formatter = logging.Formatter('[%(asctime)s] %(levelname)s: %(message)s')
    log_handler.setFormatter(formatter)
    real_logger = logging.getLogger("log")
    real_logger.addHandler(log_handler)
    real_logger.setLevel(logging.INFO)

    def frontend_callbacks_factory():
        def input_callback(data):
            real_logger.info('[input_callback] Data arrived: {!r}'.format(data))
            message = data.decode()
            return [int(x) for x in message.split()]

        def output_callback(answer):
            message = str(answer).encode()
            real_logger.info('[output_callback] Data sent: {!r}'.format(message))
            return message

        return input_callback, output_callback

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
                real_logger.info('[answer_reducer] New value of task is {}'.format(current_sum))

            def final_answer():
                return current_sum / size

            return generate_new_task, answer_reducer, final_answer

    def worker_task_handler(x):
        return x**2
    server = Server.run_server(
        frontend_server_connection_info=('127.0.0.1', 6000),
        frontend_server_callback_factory=frontend_callbacks_factory,
        task_solver_func_factory=average_task_solver_functions_factory,
        worker_task_handler=worker_task_handler,
        log=real_logger
    )
