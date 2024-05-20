cd main

echo "Build"

go clean
go build mrmaster.go || exit 1
go build mrworker.go || exit 1

cd ../apps
go build -buildmode=plugin wordcount.go || exit 1

cd ../main

/home/gentle/projects/homeworks/distribute_system/hadoop-2.10.1/bin/hdfs dfs -rm -r /mr

echo "Run Master"

./mrmaster input.txt &

sleep 1s
echo "Run Worker"

./mrworker ../apps/wordcount.so &

wait
