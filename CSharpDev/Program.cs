using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace TspSolver
{
    class Program
    {
        public static Task test_case_task()
        {
            int n = 5;
            MatrixType matrix = new MatrixType(new int[]{ Constants.POSITIVE_INF,           25,           40,           31,           27,
                       5, Constants.POSITIVE_INF,           17,           30,           25,
                      19,           15, Constants.POSITIVE_INF,            6,            1,
                       9,           50,           24, Constants.POSITIVE_INF,            6,
                      22,            8,            7,           10, Constants.POSITIVE_INF }, n);
            int[] mapping = new int[n];
            for (int i = 0; i < n; ++i)
            {
                mapping[i] = i;
            }
            return new Task(matrix, mapping, mapping, null, 0, Constants.POSITIVE_INF);
        }

        public static void run_test_case(Solver solver)
        {
            Console.WriteLine("test begin");
            Task task = test_case_task();
            Console.WriteLine("Before:");
            solver.print_square_matrix(task.Matrix);
            Console.WriteLine("After:");
            //ZeroInfo hz = solver.find_heaviest_zero(task.Matrix);
            //Console.WriteLine("Row = "+hz.Row.ToString()+"; Col = "+hz.Col.ToString()+"; W = "+ hz.Weight.ToString());
            //solver.calculate_plus_cost(task.Matrix);
            //solver.print_square_matrix(task.Matrix);
            Answer answer = solver.SolveImpl(task);
            Console.WriteLine("Answer:");
            Console.WriteLine("Cost: "+answer.Cost.ToString());
            Console.WriteLine("Jumps: " + TspSerializer.JumpArrayToXMLString(answer.Jumps));
            Console.WriteLine("test end");
        }

        static void Main(string[] args)
        {
            Solver solver = new Solver();
            run_test_case(solver);
            Console.ReadKey();
        }
    }
}
