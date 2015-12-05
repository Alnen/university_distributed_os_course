import logging
import sys
import socket
import time
import random

__author__ = 'Alex Uzhegov'


def generate_task(size):
    return " ".join((str(random.randint(0, 100)) for x in range(size)))


class Client:
    def __init__(self, *args, log):
        self.server_ip = "127.0.0.1"
        self.server_port = 6000
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
        self.log.info("[CLIENT] Received answer = {}. It took {} seconds.".format(resp.decode(), t2-t1))
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
    client = Client(log=real_logger)
    client.run()
