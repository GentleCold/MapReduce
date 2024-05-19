package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

// Add your RPC definitions here.

type AskTask struct {
	ID WorkID
}

// ask task info from master
type ReplyTask struct {
	Success    bool
	IsMap      bool
	InputFiles []string
	NReduce    int
	ID         TaskID // map / reduce id
}

type FinishTask struct {
	IsMap        bool
	Intermediate map[int]string
	ID           TaskID
}

type AskArgs struct {
}

type ReplyArgs struct {
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func masterSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
