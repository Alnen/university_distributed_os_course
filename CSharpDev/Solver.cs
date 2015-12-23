using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace TspSolver
{
    public class Solver
    {
        private System.IO.StreamWriter outfile;

        public void print_square_matrix(MatrixType matrix)
        {
            for (int i = 0; i < matrix.Size; ++i)
            {
                for (int j = 0; j < matrix.Size; ++j)
                {
                    if (matrix.Values[i * matrix.Size + j] == Constants.POSITIVE_INF)
                    {
                        outfile.Write(String.Format("{0,5:0}", "INF"));
                    }
                    else
                    {
                        outfile.Write(String.Format("{0,5:0}", matrix.Values[i * matrix.Size + j]));
                    }
                }
                outfile.WriteLine("");
            }
        }

        public Solver()
        {
            outfile = new System.IO.StreamWriter(@"W:\CSharpProjects\TspSolver\TspSolver\res.txt");
        }

        public Tuple<bool, int> calculate_plus_cost(MatrixType matrix)
        {
	        int cost = 0;
	        // every row
	        for (int i = 0; i < matrix.Size; ++i)
            {
                int min_value = Constants.POSITIVE_INF;
                // looking min value in row
                int infinity_count = 0;
		        for (int j = 0; j < matrix.Size; ++j)
                {
			        if (matrix.Values[i * matrix.Size + j] == Constants.POSITIVE_INF)
                    {
                        ++infinity_count;
			        }
                    else if (matrix.Values[i * matrix.Size + j] < min_value)
                    {
                        min_value = matrix.Values[i * matrix.Size + j];
                    }
		        }
		        // if all elements in row are infinite then return
		        if (infinity_count == matrix.Size)
                {
			        return Tuple.Create(false, 0);
		        }
		        if (min_value != 0)
                {
			        // subtract min value from entire row
			        for (int j = 0; j < matrix.Size; ++j)
                    {
				        if (matrix.Values[i * matrix.Size + j] != Constants.POSITIVE_INF)
                        {
                            matrix.Values[i * matrix.Size + j] -= min_value;
				        }
			        }
                    // add min value to cost
                    cost += min_value;
		        }
	        }
	        // every collum
	        for (int i = 0; i < matrix.Size; ++i)
            {
                int min_value = Constants.POSITIVE_INF;
                // looking min value in collum
                int infinity_count = 0;
		        for (int j = 0; j < matrix.Size; ++j)
                {
			        if (matrix.Values[j * matrix.Size + i] == Constants.POSITIVE_INF)
                    {
                        ++infinity_count;
			        }
                    else if (matrix.Values[j * matrix.Size + i] < min_value)
                    {
                        min_value = matrix.Values[j * matrix.Size + i];
			        }
		        }
		        // if all elements in collum are infinite then return
		        if (infinity_count == matrix.Size)
                {
			        return Tuple.Create(false, 0);
                }
		        if (min_value != 0)
                {
			        // subtract min value from entire column
			        for (int j = 0; j < matrix.Size; ++j)
                    {
				        if (matrix.Values[j * matrix.Size + i] != Constants.POSITIVE_INF)
                        {
                            matrix.Values[j * matrix.Size + i] -= min_value;
				        }
			        }
                    // add min value to cost
                    cost += min_value;
		        }
	        }
            return Tuple.Create(true, cost);
        }

        public ZeroInfo find_heaviest_zero(MatrixType matrix)
        {
            ZeroInfo heaviest_zero = new ZeroInfo(0, 0, Constants.NEGATIVE_INF);
	        for (int i = 0; i < matrix.Size; ++i)
            {
		        // for every zero in row
		        for (int j = 0; j < matrix.Size; ++j)
                {
			        if (matrix.Values[i * matrix.Size + j] == 0)
                    {
                        // find min element in row
                        int weight = 0;
				        int min_value = Constants.POSITIVE_INF;
				        for (int z = 0; z < matrix.Size; ++z)
                        {
					        if ((z != j) && (matrix.Values[i * matrix.Size + z] < min_value) &&
                                (matrix.Values[i * matrix.Size + z] != Constants.POSITIVE_INF))
                            {
                                min_value = matrix.Values[i * matrix.Size + z];
					        }
				        }
				        if (min_value != Constants.POSITIVE_INF)
                        {
                            weight += min_value;
				        }
                        // find min element in collum
                        min_value = Constants.POSITIVE_INF;
				        for (int z = 0; z < matrix.Size; ++z)
                        {
					        if ((z != i) && (matrix.Values[z * matrix.Size + j] < min_value) &&
                                (matrix.Values[z * matrix.Size + j] != Constants.POSITIVE_INF))
                            {
                                min_value = matrix.Values[z * matrix.Size + j];
					        }
				        }
				        if (min_value != Constants.POSITIVE_INF)
                        {
                            weight += min_value;
				        }
				        // remember new zero
				        if (heaviest_zero.Weight < weight)
                        {
                            heaviest_zero = new ZeroInfo(i, j, weight);
				        }
			        }
		        }
	        }
            return heaviest_zero;
        }

        public Tuple<Jump, bool> find_previous_jump(Jump[] all_jumps, Jump current_jump)
        {
	        for (int i = 0; i < all_jumps.Length; ++i)
            {
		        if (all_jumps[i].Destination == current_jump.Source)
                {
			        return Tuple.Create(all_jumps[i], true);
		        }
	        }
            return Tuple.Create(current_jump, false);
        }

        public Tuple<Jump, bool> find_next_jump(Jump[] all_jumps, Jump current_jump)
        {
	        for (int i = 0; i < all_jumps.Length; ++i)
            {
		        if (all_jumps[i].Source == current_jump.Destination)
                {
			        return Tuple.Create(all_jumps[i], true);
                }
	        }
	        return Tuple.Create(current_jump, false);
        }

        public Tuple<Jump, bool> find_last_jump_in_direction(Jump[] all_jumps, Jump current_jump, Direction direction)
        {
            bool found_new = false;
            Jump last_jump = current_jump;
            while (true)
            {
                switch (direction)
                {
                    case Direction.FORWARD_DIR:
                        var tuple1 = find_next_jump(all_jumps, last_jump);
                        last_jump = tuple1.Item1;
                        found_new = tuple1.Item2;
                        break;
                    case Direction.BACKWARD_DIR:
                        var tuple2 = find_previous_jump(all_jumps, last_jump);
                        last_jump = tuple2.Item1;
                        found_new = tuple2.Item2;
                        break;
                    default:
                        Console.WriteLine("[find_last_jump_in_direction] Error: No right direction!");
                        return Tuple.Create(new Jump(-1, -1), false);
                }
                if (found_new == false)
                {
                    break;
                }
            }
            if ((current_jump.Source != last_jump.Source) && (current_jump.Destination != last_jump.Destination))
            {
                return Tuple.Create(last_jump, true);
            }
            else
            {
                return Tuple.Create(last_jump, false);
            }
        }

        public void forbid_jump_if_needed(MatrixType matrix, int[] row_mapping, int[] col_mapping, Jump[] all_jumps)
        {
            int inf_x = 0, inf_y = 0;
            Jump[] jumps = new Jump[all_jumps.Length - 1];
            for (int i = 1; i < all_jumps.Length; ++i)
            {
                jumps[i - 1] = all_jumps[i];
            }
            var tuple1 = find_last_jump_in_direction(jumps, all_jumps[0], Direction.BACKWARD_DIR);
            var tuple2 = find_last_jump_in_direction(jumps, all_jumps[0], Direction.FORWARD_DIR);
            Jump beginning_of_chain = tuple1.Item1;
            bool found_next = tuple1.Item2;
            Jump end_of_chain = tuple2.Item1;
            bool found_prev = tuple2.Item2;
            outfile.WriteLine("SolveImpl 3.1.1 {" +
                beginning_of_chain.Source.ToString() + ";" +
                beginning_of_chain.Destination.ToString() + "} " +
                found_next.ToString() + " {" +
                end_of_chain.Source.ToString() + ";" +
                end_of_chain.Destination.ToString() + "} " +
                found_prev.ToString());
            if (!(found_prev && found_next))
            {
                outfile.WriteLine("cond 1");
                for (int i = 0; i < matrix.Size; ++i)
                {
                    if (row_mapping[i] == end_of_chain.Destination)
                    {
                        inf_x = i;
                    }
                    if (col_mapping[i] == beginning_of_chain.Source)
                    {
                        inf_y = i;
                    }
                }
                outfile.WriteLine("row = " + inf_x.ToString() + ", col = " + inf_y.ToString());
                matrix.Values[inf_x * matrix.Size + inf_y] = Constants.POSITIVE_INF;
            }
            else
            {
                outfile.WriteLine("cond 2");
            }
        }

        public SubTaskData generate_sub_task_data(MatrixType matrix, int[] row_mapping, int[] col_mapping, Jump[] all_jumps, ZeroInfo heaviest_zero)
        {
            MatrixType new_matrix = new MatrixType(matrix.Size - 1);
            int[] new_row_mapping = new int[matrix.Size - 1];
            int[] new_col_mapping = new int[matrix.Size - 1];
            Jump[] new_all_jumps = new Jump[ (all_jumps != null) ? all_jumps.Length + 1 : 1];
            int zero_row = heaviest_zero.Row;
            int zero_col = heaviest_zero.Col;
	        new_all_jumps[0] = new Jump(row_mapping[zero_row], col_mapping[zero_col]);
            if (all_jumps != null)
            {
                for (int j = 0; j < all_jumps.Length; ++j)
                {
                    new_all_jumps[j + 1] = all_jumps[j];
                }
            }
            // prepare new x_mapping, y_mapping and matrix.
            int i_row = 0, i_col = 0;
	        for (int i = 0; i < matrix.Size; ++i)
            {
		        if (i != zero_row)
                {
                    new_row_mapping[i_row] = row_mapping[i];
                    int j_col = 0;
			        for (int j = 0; j < matrix.Size; ++j)
                    {
				        if (j == zero_col)
                        {
                            continue;
				        }
                        new_matrix.Values[i_row * (matrix.Size - 1) + j_col] = matrix.Values[i * matrix.Size + j];
                        ++j_col;
			        }
		        }
                else
                {
                    --i_row;
		        }
		        if (i != zero_col)
                {
                    new_col_mapping[i_col] = col_mapping[i];
		        }
                else
                {
                    --i_col;
		        }
                ++i_row;
                ++i_col;
	        }
            //fmt.Printf("[generate]** \n")
            //outfile.WriteLine("SolveImpl 3.1");
            //print_square_matrix(new_matrix);
            forbid_jump_if_needed(new_matrix, new_row_mapping, new_col_mapping, new_all_jumps);
            outfile.WriteLine("SolveImpl 3.2");
            print_square_matrix(new_matrix);
            return new SubTaskData(new_matrix, new_row_mapping, new_col_mapping, new_all_jumps);
        }

        public Answer SolveImpl(Task task)
        {
	        if ((task.Matrix.Size == 0) || (task.Matrix.Size == 1)) {
		        return Constants.ERROR_ANSWER;
            }
            outfile.WriteLine("SolveImpl 1");
            print_square_matrix(task.Matrix);
            var tuple1 = calculate_plus_cost(task.Matrix);
            bool success = tuple1.Item1;
            int additional_cost = tuple1.Item2;
            if (!success)
            {
                return Constants.ERROR_ANSWER;
            }
            task.CurrCost += additional_cost;
            ZeroInfo heaviest_zero = find_heaviest_zero(task.Matrix);
            outfile.WriteLine("SolveImpl 2");
            print_square_matrix(task.Matrix);
            // if task.Size is 2 end recursion
            if (task.Matrix.Size == 2)
            {
                int next_city = heaviest_zero.Col;
                int previous_city = heaviest_zero.Row;
		        if ((task.Matrix.Values[(previous_city ^ 1)*2+(next_city^1)] == 0) &&
			        (task.Matrix.Values[(previous_city ^ 1)*2+next_city] == Constants.POSITIVE_INF) &&
			        (task.Matrix.Values[previous_city*2+(next_city^1)] == Constants.POSITIVE_INF))
                {
                    Jump[] jumps = new Jump[task.Jumps.Length+2];
                    jumps[0] = new Jump(task.RowMapping[heaviest_zero.Row], task.ColMapping[heaviest_zero.Col]);
                    jumps[1] = new Jump(task.RowMapping[heaviest_zero.Row ^ 1], task.ColMapping[heaviest_zero.Col ^ 1]);
			        for (int j = 0; j < task.Jumps.Length; ++j)
                    {

                        jumps[j + 2] = task.Jumps[j];
			        }
                    return new Answer(jumps, task.CurrCost);
		        } else {
                    return Constants.ERROR_ANSWER;
                }
	        }
            // prepare data for recursive call
            outfile.WriteLine("SolveImpl 3");
            print_square_matrix(task.Matrix);
            SubTaskData sub_dt = generate_sub_task_data(task.Matrix, task.RowMapping, task.ColMapping, task.Jumps, heaviest_zero);
            //outfile.WriteLine("SolveImpl 3.5");
            //print_square_matrix(sub_dt.Matrix);
            //call this function recursively
            Answer answer = SolveImpl(new Task(sub_dt.Matrix, sub_dt.RowMapping, sub_dt.ColMapping, sub_dt.Jumps, task.CurrCost, task.MinCost));
            //print_answer(answer, "GOT ANSWER")
            Jump[] final_path = null;
	        if (answer.Cost < task.MinCost)
            {
                final_path = answer.Jumps;
                task.MinCost = answer.Cost;
	        }
            outfile.WriteLine("SolveImpl 4");
            print_square_matrix(task.Matrix);
            //right_path
            if ((task.CurrCost + heaviest_zero.Weight) < task.MinCost)
            {
                //correct first one
                task.Matrix.Values[heaviest_zero.Row * task.Matrix.Size + heaviest_zero.Col] = Constants.POSITIVE_INF;
                answer = SolveImpl(task);
		        if (answer.Cost < task.MinCost)
                {
                    final_path = answer.Jumps;
                    task.MinCost = answer.Cost;
		        }
	        }
            outfile.WriteLine("SolveImpl 5");
            // return answer
            if (task.MinCost < Constants.POSITIVE_INF)
            {
                return new Answer(final_path, task.MinCost);
	        } else {
		        return Constants.ERROR_ANSWER;
	        }
        }

        Tuple<bool, Task, Task>  CrusherImpl(Task task)
        {
            var tuple1 = calculate_plus_cost(task.Matrix);
            bool success = tuple1.Item1;
            int additional_cost = tuple1.Item2;
            if (!success) 
            {
                return Tuple.Create(false, new Task(), new Task());
            }
            task.CurrCost += additional_cost;
            ZeroInfo heaviest_zero = find_heaviest_zero(task.Matrix);
            //prepare data for recursive call
            SubTaskData sub_dt = generate_sub_task_data(task.Matrix, task.RowMapping, task.ColMapping, task.Jumps, heaviest_zero);
            //call this function recursively
            Task task1 = new Task(sub_dt.Matrix, sub_dt.RowMapping, sub_dt.ColMapping, sub_dt.Jumps, task.CurrCost, task.MinCost);
            task.Matrix.Values[heaviest_zero.Row * task.Matrix.Size + heaviest_zero.Col] = Constants.POSITIVE_INF;
            return Tuple.Create(true, task1, task);
        }
    }
}
