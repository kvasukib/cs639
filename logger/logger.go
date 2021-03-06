package logger

import(
	"bytes"
	"exec"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type TaskId int64
type taskInfo struct{
	TaskName string
	StartTime int64
	EndTime int64
}

var logFile *os.File
var taskMap = map[TaskId] taskInfo{}
var currTaskId TaskId
var statusDir string
var currVals []float64
var currValIndex int
var notFull bool
var startId TaskId

const STATUS_LEN = 82
const STATUS_CMD = "./stats.sh"
const WHO = "./who.sh"
const WHO_LEN = 3
const MEM_FREE = "./mem.sh"
const MEM_LEN = 5
const MEM_TOTAL = "./mem_total.sh"

func Init(Filename string, Directory string) os.Error{
	//open the file we've been told
	var err os.Error
	logFile, err = os.Open(Filename, os.O_CREATE | os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("logger: unable to init log file: " + err.String());
		return err;
	}
	statusDir = Directory
	//initialize task id numbers
	currTaskId = 0;
	currVals = make([]float64, 6)
    for i:= 0; i < 5; i ++ {
       currVals[i] = 0
    }
	currValIndex = 0
	notFull = true
	startId = 0
	return nil;
}

func QuickInit() {
	currTaskId = 0;
	currVals = make([]float64, 5)
    for i:= 0; i < 5; i ++ {
       currVals[i] = 0
    }
	currValIndex = 0
	notFull = true
	startId = 0
}

func Start(TaskName string) TaskId{
	//get the start time before anything else, for consistency sake
	thisStartTime := time.Nanoseconds()
	var info taskInfo;
	info.StartTime = thisStartTime
	info.TaskName = TaskName
	taskMap[currTaskId] = info
	//make sure you don't overwrite another task
    retval := currTaskId
	currTaskId ++
	return retval
}

func End(thisTask TaskId, SysStats bool) string{
	//get the end time before anything else
	thisEndTime := time.Nanoseconds()
	info, present := taskMap[thisTask]
	if !present{
		return "logger: taskId is not in use"
	}
	info.EndTime = thisEndTime
	taskMap[thisTask] = info
	_, err := logFile.WriteString(String(thisTask))
	if err != nil {
		return err.String()
	}
	if SysStats {
		//systemStats()
	}	
	return ""
}

func systemStats() {
        args := make([]string, 1)
        var result []byte
        result = make([]byte, STATUS_LEN)
        command, err := exec.Run(STATUS_CMD, args, nil, statusDir, exec.PassThrough, exec.Pipe, exec.PassThrough)
        if err != nil{
                log.Println("chunk fails in command:" + err.String())
                log.Fatal("chunk: unable to obtain remote command")
        }
	err = nil
	
        time.Sleep(2100000000)
        _,err =command.Stdout.Read(result)
	if err != nil{
        	log.Println("chunk fails read from command: " + err.String())
	}
	logFile.Write(result)
}

func String(thisTask TaskId) string {
	var ret string;
	info := taskMap[thisTask]
	if info.EndTime == 0 {
		return ""
	}
	timeSpent := info.EndTime - info.StartTime
	niceTimeSpent := float64(timeSpent) / float64(1000000000)
	timeStamp := time.SecondsToLocalTime(time.Seconds())
	ret = "\n" + timeStamp.String() + ": " + info.TaskName + ": " + fmt.Sprintf("%f", niceTimeSpent) + " seconds\n"
	return ret
}

/********************************************************
 * Load score:
 * x/5pts for memory usage, so it would be usage % * 5
 * x/5pts for call time
*********************************************************/
func GetLoad() int {
	//first, get memory usage
	load := 5.0 * getMem()
	log.Println("logger: done getting MEM in logger");
	load += 5.0 * getCallTime()
	log.Println("logger: done getting CALLTIME in logger");
	return int(load)
}

func getMem() float64 {
	memFile, err := os.Open("/proc/meminfo", os.O_RDONLY, 0)
	if err != nil {
		log.Println("Error opening meminfo:" +  err.String())
	}
	info := make([]byte, 512)
	_, err = memFile.Read(info)
	if err != nil {
		log.Println("Error reading meminfo:" + err.String())
	}
	infoString := string(info)
	tokens := strings.Split(infoString, "\n", 3)
	return memToFloat(tokens[1]) / memToFloat(tokens[0])
}

func memToFloat(memString string) float64 {
	exp, err := regexp.Compile("[0-9]+")
	resultString := exp.FindString(memString)
	result, err := strconv.Atof64(resultString);
	if err != nil {
		log.Println(err.String());
	}
	return result
}

func getCallTime() float64 {
	lastId := currTaskId - 1
	ourId := startId;
	var retVal float64;
	for notFull && (ourId <= lastId) { 
		//log.Println("In initial phase ...")
		info, _ := taskMap[ourId]
		if info.EndTime != 0 && info.TaskName == "Write" {
			timeSpent := info.EndTime - info.StartTime
			niceTimeSpent := float64(timeSpent) / float64(1000000000)
			currVals[currValIndex] = niceTimeSpent
			//log.Println("Adding: " + fmt.Sprintf("%f", niceTimeSpent))
			currValIndex ++;
			if currValIndex >= 5 {
				//log.Println("flipping notFul1!")
				notFull = false
				currValIndex = 0
			}
			ourId++
		} else {
			ourId ++
		}
	}	
	retVal = 0
	ourId = startId
	needSample := !notFull
    	//log.Println("sample: " + fmt.Sprintf("%b", needSample) + " " + fmt.Sprintf("%d", ourId) + " " + fmt.Sprintf("%d", lastId));
	for needSample && (ourId <= lastId) {
		//log.Println("In checking phase")
        	//log.Println("sample: " + fmt.Sprintf("%b", needSample) + " " + fmt.Sprintf("%d", ourId) + " " + fmt.Sprintf("%d", lastId));
		info, _ := taskMap[ourId]
		if info.EndTime != 0 && info.TaskName == "Write" {
			timeSpent := info.EndTime - info.StartTime
			niceTimeSpent := float64(timeSpent) / float64(1000000000)
			needSample = false
			avg := (currVals[0] + currVals[1] + currVals[2] + currVals[3] + currVals[4])/5
			//log.Println("avg: " + fmt.Sprintf("%f", avg) + " niceTime: " + fmt.Sprintf("%f", niceTimeSpent))
			if (niceTimeSpent - avg) > 0 {
				retVal = (niceTimeSpent - avg) / avg
			}
			currVals[currValIndex] = niceTimeSpent
			currValIndex ++
			if currValIndex >= 5 {
				currValIndex = 0
			}
			ourId++;
		} else {
			ourId ++;
		}
	}
	startId = lastId
	//log.Println("retVal of call: " + fmt.Sprintf("%f", retVal))
    if retVal > 1.0 {
       retVal = 1.0
    }
	return retVal
}
		
			
	
	

/*func GetLoad() int {
	result := make([]byte, WHO_LEN)
        args := make([]string, 1)
	command, err := exec.Run(WHO, args, nil, statusDir, exec.PassThrough, exec.Pipe, exec.PassThrough)
        if err != nil{
                log.Println("logger fails in command:" + err.String())
                log.Fatal("logger: unable to obtain remote command")
	}
        _,err =command.Stdout.Read(result)
	if err != nil{
        	log.Println("logger fails read from command: " + err.String())
	}
	whoResult := commandToInt(result)
	result = make([]byte, MEM_LEN)
	command, err = exec.Run(MEM_FREE, args, nil, statusDir, exec.PassThrough, exec.Pipe, exec.PassThrough)
        if err != nil{
                 log.Println("logger fails in command:" + err.String())
                log.Fatal("logger: unable to obtain remote command")
        }
        _,err =command.Stdout.Read(result)
	if err != nil{
        	log.Println("logger: fails read from command: " + err.String())
	}
	freeResult := commandToInt(result)
	result = make([]byte, MEM_LEN)
	command, err = exec.Run(MEM_TOTAL, args, nil, statusDir, exec.PassThrough, exec.Pipe, exec.PassThrough)
        if err != nil{
                 log.Println("logger fails in command:" + err.String())
                log.Fatal("logger: unable to obtain remote command")
        }
        _,err =command.Stdout.Read(result)
	if err != nil{
        	log.Println("logger fails read from command: " + err.String())
	}
	totalResult := commandToInt(result)
	mem_usage := (1 - (float32(freeResult) / float32(totalResult))) * 5
	user_percent := float32(whoResult) * .25
	return int(user_percent) + int(mem_usage)
}*/

func commandToInt(result []byte) int {
	index := bytes.IndexRune(result, 10)
	result2 := result[0:index]
	resultString := string(result2)
	resultInt,err := strconv.Atoi(resultString)
	if err != nil {
		log.Print(err.String())
	}
	return resultInt
}	
