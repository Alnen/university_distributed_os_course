import logging
import sys
import socket
import time
import math
import random
import numpy as np

from lxml import etree

__author__ = 'Alex Uzhegov'


def generate_task(size, *args, min_value=0, max_value=100):
    for i in range(size):
        for j in range(size):
            if i == j:
                yield np.iinfo(np.int32).max
            else:
                yield random.randint(min_value, max_value)


def deserialize_answer(answer_binary):
    root = etree.fromstring(answer_binary)
    cost = int(root[0].text)
    jumps = [tuple(jump.split('-')) for jump in root[1].text.split(',')]
    return cost, jumps


def serialize_task(matrix):
    root = etree.Element('task')

    size_element = etree.Element('size')
    size_element.text = str(int(math.sqrt(len(matrix))))
    root.append(size_element)

    matrix_element = etree.Element('matrix')
    matrix_element.text = ' '.join((str(val) for val in matrix))
    root.append(matrix_element)

    xml = etree.tostring(root, pretty_print=True, xml_declaration=True)
    print(xml.decode())
    return xml


class Client:
    def __init__(self, *args, log):
        self.server_ip = "127.0.0.1"
        self.server_port = 6000
        self.log = log

    def solve_task(self, task):
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
        task_binary = serialize_task(task)
        s.send(task_binary)
        self.log.info("[CLIENT] Sending task done. Trying to receive answer...")
        response = s.recv(4096)
        s.close()
        self.log.info("[CLIENT] Received answer.")
        return deserialize_answer(response)

    def run(self):
        task = generate_task(30)
        task_data = list(task)
        t1 = time.time()
        cost, jumps = self.solve_task(task_data)
        t2 = time.time()
        self.log.info("[CLIENT] Received answer = {}. Jumps are {}. It took {} sec.".format(cost, jumps, t2-t1))


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
