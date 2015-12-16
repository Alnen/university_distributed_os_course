package types

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type (
	TaskXML struct {
		XMLName xml.Name `xml:"task"`
		Size    int      `xml:"size"`
		Matrix  string   `xml:"matrix"`
	}
	AnswerXML struct {
		XMLName xml.Name `xml:"answer"`
		Cost    int      `xml:"cost"`
		Jumps   string   `xml:"jumps"`
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
		Matrix     *MatrixType
		RowMapping []int
		ColMapping []int
		Jumps      []JumpType
	}
	AnswerType struct {
		Jumps []JumpType
		Cost  DataType
	}
	DataTaskType struct {
		Matrix     *MatrixType
		RowMapping []int
		ColMapping []int
		Size       int
	}
	TaskType struct {
		Matrix            *MatrixType
		RowMapping        []int
		ColMapping        []int
		Jumps             []JumpType
		CurrCost, MinCost DataType
		Size              int
	}
	GlobalCostType struct {
		value DataType
		mutex chan bool
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
		&MatrixType{}, []int{}, []int{},
		[]JumpType{}, 0, 0, 0,
	}
)

// ----------------- AnswerType -----------------
func (answer *AnswerType) ToXml() []byte {
	str_jumps := ""
	for i := 0; i < len(answer.Jumps); i++ {
		if i > 0 {
			str_jumps += ","
		}
		str_jumps += strconv.Itoa(answer.Jumps[i].Source)
		str_jumps += "-" + strconv.Itoa(answer.Jumps[i].Destination)
	}
	xml_answer := &AnswerXML{Cost: int(answer.Cost), Jumps: str_jumps /*JumpTypeArrayToString(answer.Jumps)*/}
	xml_string, err := xml.MarshalIndent(xml_answer, "", "")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return []byte{}
	}
	return xml_string
}

func (answer *AnswerType) FromXml(data []byte) {
	xml_answer := AnswerXML{}
	err := xml.Unmarshal(data, &xml_answer)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	answer.Cost = DataType(xml_answer.Cost)
	answer.Jumps = JumpTypeArrayFromString(xml_answer.Jumps)
}

func (answer *AnswerType) ToString() string {
	str_data := strconv.Itoa(int(answer.Cost))
	for i := 0; i < len(answer.Jumps); i++ {
		str_data += " " + strconv.Itoa(answer.Jumps[i].Source)
		str_data += " " + strconv.Itoa(answer.Jumps[i].Destination)
	}
	return str_data
}

func (answer *AnswerType) FromString(data string) {
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

// ----------------- TaskType -----------------
func (task *TaskType) ToXml() []byte {
	xml_task := &TaskXML{Size: task.Size, Matrix: task.Matrix.ToString()}
	xml_string, err := xml.MarshalIndent(xml_task, "", "")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return []byte{}
	}
	return xml_string
}

func (task *TaskType) FromXml(data []byte) {
	xml_task := TaskXML{}
	//fmt.Printf("BEFOR UNMARSHAL\n")
	err := xml.Unmarshal(data, &xml_task)
	//fmt.Println("AFTER  UNMARSHAL", xml_task.Size)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	//fmt.Printf("1\n")
	matrix := make(MatrixType, xml_task.Size*xml_task.Size)
	//fmt.Printf("2\n")
	matrix.FromString(xml_task.Matrix, xml_task.Size)
	//fmt.Printf("2\n")
	mapping := make([]int, xml_task.Size)
	for i := 0; i < xml_task.Size; i++ {
		mapping[i] = i
	}
	//fmt.Printf("3\n")
	task.Matrix = &matrix
	task.RowMapping = mapping
	task.ColMapping = mapping
	task.Jumps = nil
	task.CurrCost = DataType(0)
	task.MinCost = DataType(POSITIVE_INF)
	task.Size = xml_task.Size
	//fmt.Println("4", task.Size, len(task.Matrix))
}

func (task *TaskType) ToString() string {
	size := task.Size
	str_data := strconv.Itoa(size)
	matrix := *task.Matrix
	for i := 0; i < size*size; i++ {
		str_data += " " + strconv.Itoa(int(matrix[i]))
	}
	for i := 0; i < size; i++ {
		str_data += " " + strconv.Itoa(int(task.RowMapping[i]))
	}
	for i := 0; i < size; i++ {
		str_data += " " + strconv.Itoa(int(task.ColMapping[i]))
	}
	jump_size := len(task.Jumps)
	str_data += " " + strconv.Itoa(jump_size)
	for i := 0; i < jump_size; i++ {
		str_data += " " + strconv.Itoa(int(task.Jumps[i].Source))
		str_data += " " + strconv.Itoa(int(task.Jumps[i].Destination))
	}
	str_data += " " + strconv.Itoa(int(task.CurrCost))
	str_data += " " + strconv.Itoa(int(task.MinCost))
	return str_data
}

func (task *TaskType) FromString(data string) {
	data_vec := strings.Split(data, " ")
	vec_size := len(data_vec)
	if vec_size < 4 {
		return
	}
	size, err := strconv.Atoi(data_vec[0])
	if err != nil {
		fmt.Printf("task.FromString convert matrix len error: %v\n", err)
	}
	matrix := make(MatrixType, size*size)
	row_mapping := make([]int, size)
	col_mapping := make([]int, size)
	//fmt.Println("[FROM STRING]: data_vec.size = ", len(data_vec), " | ", (1 + (size+2)*size))
	/*
	if len(data_vec) <= (1 + (size+2)*size) {
		fmt.Printf("[FROM STRING]: size: %d\n", size)
		fmt.Printf("[FROM STRING]: data_vec: %d\n", len(data_vec))
		fmt.Printf("[FROM STRING]: data_str: \"%d\"\n", len([]byte(data)))
	}
	*/
	jumps_len, _ := strconv.Atoi(data_vec[1+(size+2)*size])
	jumps := make([]JumpType, jumps_len)
	curr_cost, err := strconv.Atoi(data_vec[vec_size-2])
	if err != nil {
		fmt.Printf("task.FromString curr_cost (%s) error: %v\n", string(data_vec[vec_size-2]), err)
	}
	min_cost, err := strconv.Atoi(data_vec[vec_size-1])
	if err != nil {
		fmt.Printf("task.FromString min_cost (%s) error: %v\n", string(data_vec[vec_size-1]), err)
	}
	offset := 1
	for i := offset; i < offset+size*size; i++ {
		matrix_value, _ := strconv.Atoi(data_vec[i])
		matrix[i-offset] = DataType(matrix_value)
	}
	offset = offset + size*size
	for i := offset; i < offset+size; i++ {
		row_mapping[i-offset], _ = strconv.Atoi(data_vec[i])
	}
	offset = offset + size
	for i := offset; i < offset+size; i++ {
		col_mapping[i-offset], _ = strconv.Atoi(data_vec[i])
	}
	offset = offset + size + 1
	j := 0
	for i := offset; i < offset+jumps_len*2; i += 2 {
		jumps[j].Source, _ = strconv.Atoi(data_vec[i])
		jumps[j].Destination, _ = strconv.Atoi(data_vec[i+1])
		j++
	}
	//fmt.Printf("[FROM STRING]: size: %d\n", size)
	//fmt.Printf("[FromString] row_mapping: %v\n", row_mapping)
	//fmt.Printf("[FromString] col_mapping: %v\n", row_mapping)
	//fmt.Printf("[FromString] Jumps: %v\n", jumps)
	//fmt.Printf("[FromString] CurrCost: %d\n", curr_cost)
	//fmt.Printf("[FromString] MinCost: %d\n", min_cost)
	task.Matrix = &matrix
	task.RowMapping = row_mapping
	task.ColMapping = col_mapping
	task.Jumps = jumps
	task.CurrCost = DataType(curr_cost)
	task.MinCost = DataType(min_cost)
	task.Size = size
	/*
		fmt.Println("Matrix size: ", len(task.Matrix))
		fmt.Println("XMapping size: ", len(task.XMapping))
		fmt.Println("YMapping size: ", len(task.YMapping))
		fmt.Printf("Jumps: %v\n", task.Jumps)
		fmt.Println("Solution_cost: ", int(task.SolutionCost))
		fmt.Println("MinCost: ", int(task.MinCost))
		fmt.Println("Size: ", task.Size)
	*/
}

// ----------------- MatrixType -----------------
func (matrix *MatrixType) ToString() string {
	str_data := ""
	for i := 0; i < len(*matrix); i++ {
		if i > 0 {
			str_data += " "
		}
		str_data += strconv.Itoa(int((*matrix)[i]))
	}
	return str_data
}

func (matrix *MatrixType) FromString(str string, size int) {
	str_vec := strings.Split(str, " ")
	var val int
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			val, _ = strconv.Atoi(str_vec[i*size+j])
			(*matrix)[i*size+j] = DataType(val)
		}
	}
}

// ------------ Jump Array convert ------------------
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

// ----- Serialize and Deserialize vector ------
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

// --------------- GlobalCostType ------------
func (ai *GlobalCostType) Init(v DataType) {
	ai.value = v
	ai.mutex = make(chan bool, 1)
	ai.mutex <- true
}

func (ai *GlobalCostType) Get() DataType {
	<-ai.mutex
	v := ai.value
	ai.mutex <- true
	return v
}

func (ai *GlobalCostType) Set(v DataType) {
	<-ai.mutex
	ai.value = v
	ai.mutex <- true
}

// ----------- COUNTER -------------------------
type CounterType struct {
	value int
	mutex chan bool
}

func (ai *CounterType) Init(v int) {
	ai.value = v
	ai.mutex = make(chan bool, 1)
	ai.mutex <- true
}

func (ai *CounterType) Get() int {
	<-ai.mutex
	v := ai.value
	ai.mutex <- true
	return v
}

func (ai *CounterType) Inc() {
	<-ai.mutex
	ai.value++
	ai.mutex <- true
}

//----------------------------------------------
