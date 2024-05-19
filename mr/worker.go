package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"sort"

	"github.com/colinmarc/hdfs/v2"
)

const masterAddress = "127.0.0.1"

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// 1. register to master
	var workID WorkID
	fakeArgs := AskArgs{}
	fakeReply := ReplyArgs{}

	ok := call("Master.RegisterWorker", &fakeArgs, &workID)
	if !ok {
		return
	}

	// 2. ask for task
	client, e := hdfs.New(masterAddress + ":9000")
	if e != nil {
		log.Fatal("hdfs error:", e)
	}

	for {
		task := ReplyTask{}
		ok := call("Master.ApplyTask", &AskTask{workID}, &task)
		if ok {
			if task.Success == false {
				continue
			}
			// 3. handle task
			intermediate := make(map[int]string)
			if task.IsMap {
				for _, filename := range task.InputFiles {
					// open and get content
					file, e := client.Open(filename)
					if e != nil {
						log.Fatal("hdfs open file error:", e)
					}
					content, e := io.ReadAll(file)
					if e != nil {
						log.Fatal("hdfs read file error:", e)
					}
					file.Close()

					// create intermediate file
					kva := mapf(filename, string(content))
					for _, kv := range kva {
						reduce := ihash(kv.Key) % task.NReduce
						filename := fmt.Sprintf("/mr/mr-%v-%v-%v", task.ID, reduce, workID)

						if e != nil {
							log.Fatal("hdfs error:", e)
						}

						var file *hdfs.FileWriter
						if _, ok := intermediate[reduce]; !ok {
							intermediate[reduce] = filename
							file, _ = client.Create(filename)
						} else {
							file, _ = client.Append(filename)
						}

						enc := json.NewEncoder(file)
						enc.Encode(&kv)
						file.Close()
					}
				}
			} else {
				kva := []KeyValue{}

				// read all kv
				for _, filename := range task.InputFiles {
					// open and get content
					file, err := client.Open(filename)
					if err != nil {
						log.Fatalf("cannot open %v", filename)
					}
					dec := json.NewDecoder(file)
					for {
						var kv KeyValue
						if err := dec.Decode(&kv); err != nil {
							break
						}
						kva = append(kva, kv)
					}
				}

				sort.Sort(ByKey(kva))

				oname := fmt.Sprintf("/mr/mr-out-%v", task.ID)
				ofile, _ := client.Create(oname)

				i := 0
				for i < len(kva) {
					j := i + 1
					for j < len(kva) && kva[j].Key == kva[i].Key {
						j++
					}
					values := []string{}
					for k := i; k < j; k++ {
						values = append(values, kva[k].Value)
					}
					output := reducef(kva[i].Key, values)
					fmt.Fprintf(ofile, "%v %v\n", kva[i].Key, output)
					i = j
				}
				ofile.Close()
			}

			// 4. finish the task and tell the master, then ask for task again
			ok := call("Master.FinishTask", &FinishTask{task.IsMap, intermediate, task.ID}, &fakeReply)
			if !ok {
				return
			}
		} else {
			break
		}
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("tcp", masterAddress+":1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
