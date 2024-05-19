package mr

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type WorkID int
type TaskID int

type TaskInfo struct {
	IsMap      bool
	InputFiles []string
	ID         TaskID
}

type Master struct {
	NReduce int

	// track if done
	MapWg   sync.WaitGroup
	MapDone bool

	ReduceWg   sync.WaitGroup
	ReduceDone bool

	// task info
	Tasks chan TaskInfo

	// protect the shared var
	Mu sync.Mutex

	Intermediate map[int][]string

	// worker info
	NextWorkerID WorkID

	//  black list of workers
	WorkerTrack map[WorkID]bool
	TaskTrack   [2]map[TaskID]bool
}

// turn bool to int
func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (c *Master) RegisterWorker(args *AskArgs, reply *WorkID) error {
	c.Mu.Lock()
	*reply = c.NextWorkerID
	c.WorkerTrack[*reply] = true
	c.NextWorkerID++
	c.Mu.Unlock()
	return nil
}

func (c *Master) ApplyTask(args *AskTask, reply *ReplyTask) error {
	t, ok := <-(c.Tasks)
	if !ok {
		return errors.New("no more task")
	}

	c.Mu.Lock()
	if c.TaskTrack[Btoi(t.IsMap)][t.ID] {
		c.Mu.Unlock()
		reply.Success = false
		return nil
	}
	c.TaskTrack[Btoi(t.IsMap)][t.ID] = false
	c.Mu.Unlock()

	reply.Success = true
	reply.IsMap = t.IsMap
	reply.InputFiles = t.InputFiles
	reply.ID = t.ID
	reply.NReduce = c.NReduce

	go func(t TaskInfo) {
		time.Sleep(10 * time.Second)
		c.Mu.Lock()
		defer c.Mu.Unlock()
		if c.TaskTrack[Btoi(t.IsMap)][t.ID] {
			return
		}
		c.Tasks <- t
	}(t)

	return nil
}

func (c *Master) FinishTask(args *FinishTask, reply *ReplyArgs) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.TaskTrack[Btoi(args.IsMap)][args.ID] {
		return nil
	}

	if args.IsMap {
		// record the Intermediate file
		for k, v := range args.Intermediate {
			c.Intermediate[k] = append(c.Intermediate[k], v)
		}
		c.MapWg.Done()
	} else {
		c.ReduceWg.Done()
	}

	c.TaskTrack[Btoi(args.IsMap)][args.ID] = true

	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Master) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	// l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func (c *Master) Done() bool {
	if c.ReduceDone {
		close(c.Tasks)
	}
	return c.ReduceDone
}

func MakeMaster(files []string, nReduce int) *Master {
	nMap := len(files)

	c := Master{
		NReduce:      nReduce,
		Intermediate: make(map[int][]string),
		Tasks:        make(chan TaskInfo, nMap),
		WorkerTrack:  make(map[WorkID]bool),
		TaskTrack:    [2]map[TaskID]bool{make(map[TaskID]bool), make(map[TaskID]bool)},
	}

	// record task num
	c.MapWg.Add(nMap)
	c.ReduceWg.Add(nReduce)

	// generate map task
	var i TaskID
	for f := range files {
		c.Tasks <- TaskInfo{true, []string{files[f]}, i}
		i++
	}

	// background task to wait for completion of map and reduce
	go func(c *Master) {
		c.MapWg.Wait()
		c.MapDone = true

		for i := 0; i < c.NReduce; i++ {
			c.Tasks <- TaskInfo{false, c.Intermediate[i], TaskID(i)}
		}

		c.ReduceWg.Wait()
		c.ReduceDone = true
	}(&c)

	c.server()
	return &c
}
