using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using System.Xml;
using System.IO;
using System.Text;

namespace TspSolver
{
    public struct ZeroInfo
    {
        public int Row;
        public int Col;
        public int Weight;

        public ZeroInfo(int row = 0, int col = 0, int weight = 0)
        {
            Row = row;
            Col = col;
            Weight = weight;
        }
    }

    public struct Jump
    {
        public int Source;
        public int Destination;

        public Jump(int s = -1, int d = -1)
        {
            Source = s;
            Destination = d;
        }
    }

    public struct MatrixType
    {
        public int[] Values;
        public int Size;

        public MatrixType(int size)
        {
            Values = new int[size*size];
            Size = size;
        }

        public MatrixType(int[] values = null, int size = 0)
        {
            Values = values;
            Size = size;
        }

        public override string ToString()
        {
            return "";
        }

        public void FromString(string str, int size)
        {
        }
    }

    public struct SubTaskData
    {
        public MatrixType Matrix;
        public int[] RowMapping;
        public int[] ColMapping;
        public Jump[] Jumps;

        public SubTaskData(MatrixType matrix = new MatrixType(), int[] row_mapping = null, int[] col_mapping = null, Jump[] jumps = null)
        {
            Matrix = matrix;
            RowMapping = row_mapping;
            ColMapping = col_mapping;
            Jumps = jumps;
        }
    }

    public struct Answer
    {
        public Jump[] Jumps;
        public int Cost;

        public Answer(int cost = 0)
        {
            Jumps = null;
            Cost = cost;
        }

        public Answer(Jump[] jumps, int cost = 0)
        {
            Jumps = jumps;
            Cost = cost;
        }

        public Byte[] ToXml()
        {
            return null;
	    }

        public void FromXml(Byte[] data)
        {
            return;
        }

        public override string ToString()
        {
            return "";
        }

        public void FromString(string data)
        {
        }
    }

    public struct TaskData
    {
        public MatrixType Matrix;
        public int[] RowMapping;
        public int[] ColMapping;
    }

    public struct Task
    {
        public MatrixType Matrix;
        public int[] RowMapping;
        public int[] ColMapping;
        public Jump[] Jumps;
        public int CurrCost;
        public int MinCost;

        public Task(int[] matrix = null, int size = 0)
        {
            Matrix = new MatrixType(matrix, size);
            RowMapping = null;
            ColMapping = null;
            Jumps = null;
            CurrCost = 0;
            MinCost = 0;
        }

        public Task(MatrixType matrix, int[] row_mapping, int[] col_mapping, Jump[] jumps, int curr_cost, int min_cost)
        {
            Matrix = matrix;
            RowMapping = row_mapping;
            ColMapping = col_mapping;
            Jumps = jumps;
            CurrCost = curr_cost;
            MinCost = min_cost;
        }

        public Byte[] ToXml()
        {
            return null;
        }

        public void FromXml(Byte[] data)
        {
            return;
        }

        public override string ToString()
        {
            return "";
        }

        public void FromString(string data)
        {
        }
    }

    public enum Direction { FORWARD_DIR = 1, BACKWARD_DIR = -1, UNKNOWN = 2};

    public static class Constants
    {
        public const int POSITIVE_INF = 2147483647;
        public const int NEGATIVE_INF = -2147483648;
        public const int NVAL_INDEX = -1;
        public static readonly Answer ERROR_ANSWER = new Answer(null, POSITIVE_INF);
        public static readonly Task ERROR_TASK = new Task();
    }
}
