cd main

echo "Build"

go clean
go build mrmaster.go || exit 1
go build mrworker.go || exit 1

cd ../apps
go build -buildmode=plugin wordcount.go || exit 1

cd ../main

echo "Run Master"

/home/gentle/projects/homeworks/distribute_system/hadoop-2.10.1/bin/hdfs dfs -rm /mr

./mrmaster input.txt &

echo "Run Worker"

./mrworker ../apps/wordcount.so &

wait
