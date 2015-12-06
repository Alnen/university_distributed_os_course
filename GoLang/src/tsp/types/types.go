package types

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type (
	TaskXML struct {
		XMLName      xml.Name  "xml:'task'"
		Size  int "xml:'size'"
		Matrix string "xml:'matrix'"
	}
	AnswerXML struct {
		XMLName xml.Name "xml:'answer'"
		Cost    int      "xml:'cost'"
		Jumps   string   "xml:'jumps'"
	}
)

type (
	DataType      int
	DirectionType int
	MatrixType    []DataType
	ZeroInfoType  struct {
		Row, Col int
		Weight   DataType
	}
	JumpType struct {
		Source, Destination int
	}
	SubTaskDataType struct {
		Matrix   MatrixType
		XMapping []int
		YMapping []int
		Jumps    []JumpType
	}
	AnswerType struct {
		Jumps []JumpType
		Cost  DataType
	}
	DataTaskType struct {
		Matrix   MatrixType
		XMapping []int
		YMapping []int
		Size     int
	}
	TaskType struct {
		Matrix                MatrixType
		XMapping              []int
		YMapping              []int
		Jumps                 []JumpType
		SolutionCost, MinCost DataType
		Size                  int
	}
	DivTaskType struct {
		Enabled bool
		Task1   TaskType
		Task2   TaskType
	}
)

const (
	FORWARD_DIR  DirectionType = 1
	BACKWARD_DIR DirectionType = -1
	POSITIVE_INF DataType      = 2147483647
	NEGATIVE_INF DataType      = -2147483648
	NVAL_INDEX   int           = -1
)

var (
	ERROR_ANSWER AnswerType = AnswerType{
		[]JumpType{},
		POSITIVE_INF,
	}
	ERROR_TASK TaskType = TaskType{
		MatrixType{}, []int{}, []int{},
		[]JumpType{}, 0, 0, 0,
	}
)

func (answer AnswerType) ToXml() []byte {
	xml_answer := &AnswerXML{Cost: int(answer.Cost), Jumps: JumpTypeArrayToString(answer.Jumps)}
	xml_string, err := xml.MarshalIndent(xml_answer, "", "")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return []byte{}
	}
	return xml_string
}

func (answer AnswerType) FromXml(data []byte) {
	xml_answer := AnswerXML{}
	err := xml.Unmarshal(data, &xml_answer)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	answer.Cost = DataType(xml_answer.Cost)
	answer.Jumps = JumpTypeArrayFromString(xml_answer.Jumps)
}

func (answer AnswerType) ToString() string {
	str_data := strconv.Itoa(int(answer.Cost))
	for i := 0; i < len(answer.Jumps); i++ {
		str_data += " " + strconv.Itoa(answer.Jumps[i].Source)
		str_data += " " + strconv.Itoa(answer.Jumps[i].Destination)
	}
	return str_data
}

func (answer AnswerType) FromString(data string) {
	data_vec := strings.Split(data, " ")
	vec_size := len(data_vec)
	if vec_size < 1 {
		return
	}
	jumps := make([]JumpType, (vec_size-1)/2)
	j := 0
	for i := 1; i < vec_size; i += 2 {
		jumps[j].Source, _ = strconv.Atoi(data_vec[i])
		jumps[j].Destination, _ = strconv.Atoi(data_vec[i+1])
		j++
	}
	cost, _ := strconv.Atoi(data_vec[0])
	answer.Jumps = jumps
	answer.Cost = DataType(cost)
}

func (task TaskType) ToXml() []byte {
	xml_task := &TaskXML{Size: task.Size, Matrix: task.Matrix.ToString()}
	xml_string, err := xml.MarshalIndent(xml_task, "", "")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return []byte{}
	}
	return xml_string
}

func (task TaskType) FromXml(data []byte) {
	xml_task := TaskXML{}
	err := xml.Unmarshal(data, &xml_task)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	matrix := MatrixType{}
	matrix.FromString(xml_task.Matrix, xml_task.Size)
	mapping := make([]int, xml_task.Size)
	for i := 0; i < xml_task.Size; i++ {
		mapping[i] = i
	}
	task = TaskType{matrix, mapping, mapping, []JumpType{}, DataType(0), DataType(POSITIVE_INF), xml_task.Size}
}

func (task TaskType) ToString() string {
	size := task.Size
	str_data := strconv.Itoa(size)
	for i := 0; i < size*size; i++ {
		str_data += " " + strconv.Itoa(int(task.Matrix[i]))
	}
	for i := 0; i < size; i++ {
		str_data += " " + strconv.Itoa(int(task.XMapping[i]))
	}
	for i := 0; i < size; i++ {
		str_data += " " + strconv.Itoa(int(task.YMapping[i]))
	}
	jump_size := len(task.Jumps)
	str_data += " " + strconv.Itoa(jump_size)
	for i := 0; i < jump_size; i++ {
		str_data += " " + strconv.Itoa(int(task.Jumps[i].Source))
		str_data += " " + strconv.Itoa(int(task.Jumps[i].Destination))
	}
	str_data += " " + strconv.Itoa(int(task.SolutionCost))
	str_data += " " + strconv.Itoa(int(task.MinCost))
	return str_data
}

func (task TaskType) FromString(data string) {
	data_vec := strings.Split(data, " ")
	vec_size := len(data_vec)
	if vec_size < 4 {
		return
	}
	size, _ := strconv.Atoi(data_vec[0])
	matrix := make(MatrixType, size*size)
	x_mapping := make([]int, size)
	y_mapping := make([]int, size)
	jumps_len, _ := strconv.Atoi(data_vec[1+(size+2)*size])
	jumps := make([]JumpType, jumps_len)
	solution_cost, _ := strconv.Atoi(data_vec[vec_size-2])
	min_cost, _ := strconv.Atoi(data_vec[vec_size-1])
	offset := 1
	for i := offset; i < offset+size*size; i++ {
		matrix_value, _ := strconv.Atoi(data_vec[i])
		matrix[i-offset] = DataType(matrix_value)
	}
	offset = offset + size*size
	for i := offset; i < offset+size; i++ {
		x_mapping[i-offset], _ = strconv.Atoi(data_vec[i])
	}
	offset = offset + size
	for i := offset; i < offset+size; i++ {
		y_mapping[i-offset], _ = strconv.Atoi(data_vec[i])
	}
	offset = offset + size
	j := 0
	for i := offset; i < offset+jumps_len*2; i += 2 {
		jumps[j].Source, _ = strconv.Atoi(data_vec[i])
		jumps[j].Destination, _ = strconv.Atoi(data_vec[i+1])
		j++
	}
	task = TaskType{matrix, x_mapping, y_mapping, jumps, DataType(solution_cost), DataType(min_cost), size}
}

func (matrix MatrixType) ToString() string {
	str_data := ""
	for i := 0; i < len(matrix); i++ {
		if i > 0 {
			str_data += " "
		}
		if matrix[i] == POSITIVE_INF {
			str_data += "INF"
		} else {
			str_data += strconv.Itoa(int(matrix[i]))
		}
	}
	return str_data
}

func (matrix MatrixType) FromString(str string, size int) {
	str_vec := strings.Split(str, " ")
	matrix = make(MatrixType, size*size)
	var val int
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if str_vec[i*size+j] == "INF" {
				matrix[i*size+j] = POSITIVE_INF
			} else {
				val, _ = strconv.Atoi(str_vec[i*size+j])
				matrix[i*size+j] = DataType(val)
			}
		}
	}
}

func JumpTypeArrayToString(jumps []JumpType) string {
	str_data := ""
	for i := 0; i < len(jumps); i++ {
		if i > 0 {
			str_data += " "
		}
		str_data += strconv.Itoa(jumps[i].Source)
		str_data += " " + strconv.Itoa(jumps[i].Destination)
	}
	return str_data
}

func JumpTypeArrayFromString(str string) []JumpType {
	vec_str := strings.Split(str, " ")
	jumps := make([]JumpType, len(vec_str)/2)
	j := 0
	for i := 0; i < len(vec_str); i += 2 {
		val, _ := strconv.Atoi(vec_str[i])
		jumps[j].Source = val
		val, _ = strconv.Atoi(vec_str[i+1])
		jumps[j].Destination = val
		j++
	}
	return jumps
}

func SerializeVector(vec []int) string {
	data := ""
	for i := 0; i < len(vec); i++ {
		if i > 0 {
			data += " "
		}
		data += strconv.Itoa(vec[i])
	}
	return data
}

func DeserializeVector(s string) []int {
	str_vec := strings.Split(s, " ")
	vec := make([]int, len(str_vec))
	for i := 0; i < len(str_vec); i++ {
		number, err := strconv.Atoi(str_vec[i])
		if err != nil {
			fmt.Printf("error: %v", err)
			return []int{}
		}
		vec[i] = number
	}
	return vec
}
