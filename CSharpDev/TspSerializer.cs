using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace TspSolver
{
    public class TspSerializer
    {
        public static string JumpArrayToString(Jump[] jumps)
        {
            string str = "";
            for (int i = 0; i < jumps.Length; ++i)
            {
                if (i != 0)
                {
                    str += " ";
                }
                str += jumps[i].Source + " " + jumps[i].Destination;
            }
            return str;
        }

        public static string JumpArrayToXMLString(Jump[] jumps)
        {
            string str = "";
            for (int i = 0; i < jumps.Length; ++i)
            {
                if (i != 0)
                {
                    str += ",";
                }
                str += jumps[i].Source + "-" + jumps[i].Destination;
            }
            return str;
        }

        public static Jump[] StringToJumpArray(string str)
        {
            if (str.Length < 3)
            {
                return null;
            }
            string[] str_jumps = str.Split(' ');
            Jump[] jumps = new Jump[str_jumps.Length];
            for (int i = 0, j = 0; i < str_jumps.Length; i+=2, ++j)
            {
                jumps[j] = new Jump(Int32.Parse(str_jumps[i]), Int32.Parse(str_jumps[i+1]));
            }
            return jumps;
        }
    }
}
