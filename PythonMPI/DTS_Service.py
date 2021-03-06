import logging
import sys
import Server.Server as Server
import numpy as np
import DTS_SOLVER.dts_solver as dts_solver
from struct import pack
from lxml import etree


__author__ = 'Alex Uzhegov'

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

    logger = real_logger

    def frontend_callbacks_factory():
        def input_callback(data):
            real_data = data[8:]
            logger.info('[input_callback] Data arrived: {!r}'.format(real_data))
            root = etree.fromstring(real_data)
            size = int(root[0].text)
            value_generator = iter(int(x) for x in root[1].text.split())
            return size, np.array([[next(value_generator) for j in range(size)] for i in range(size)])

        def output_callback(answer):
            cost, jumps = answer
            root = etree.Element('answer')
            size_element = etree.Element('cost')
            size_element.text = repr(int(cost))
            root.append(size_element)
            matrix_element = etree.Element('jumps')
            matrix_element.text = ",".join(str(jump.begin)+'-'+str(jump.end) for jump in jumps)
            root.append(matrix_element)
            xml = etree.tostring(root, pretty_print=True, xml_declaration=True)
            logger.info('[output_callback] Data sent: {}'.format(xml))
            msg_size = pack('<q', len(xml))
            return msg_size+xml

        return input_callback, output_callback

    def average_task_solver_functions_factory():
            min_const = dts_solver.POSITIVE_INF
            min_solutions_jumps = []

            def generate_new_task(task_data):
                nonlocal min_const
                size, matrix = task_data
                x_mapping = np.array(range(size))
                y_mapping = np.array(range(size))

                task2 = [
                    matrix,
                    x_mapping,
                    y_mapping,
                    [],
                    0,
                    min_const
                ]
                print(1)
                success, (task1, task2) = dts_solver.crusher_impl(*task2)
                print(2, success)
                while success and task1[4] < task1[5]:
                    print(3)
                    yield task1
                    task2[5] = min_const
                    success, (task1, task2) = dts_solver.crusher_impl(*task2)
                print(4)

            def answer_reducer(task_intermediate_answer):
                nonlocal min_const
                nonlocal min_solutions_jumps
                cost = task_intermediate_answer.cost
                jumps = task_intermediate_answer.jumps
                if cost < min_const:
                    min_const = cost
                    min_solutions_jumps = jumps
                logger.info('[answer_reducer] New cost of task is {}'.format(min_const))

            def final_answer():
                return min_const, min_solutions_jumps

            return generate_new_task, answer_reducer, final_answer

    def worker_task_handler(x):
        return dts_solver.solve_impl(*x)
    server = Server.run_server(
        frontend_server_connection_info=('127.0.0.1', 6000),
        frontend_server_callback_factory=frontend_callbacks_factory,
        task_solver_func_factory=average_task_solver_functions_factory,
        worker_task_handler=worker_task_handler,
        log=logger
    )
