cd main

echo "Build"

go clean
go build mrmaster.go || exit 1
go build mrworker.go || exit 1

cd ../apps
go build -buildmode=plugin wordcount.go || exit 1

cd ../main

if [ -d "tmp" ]; then
  rm -rf tmp
fi

mkdir tmp
cd tmp

echo "Run Master"

../mrmaster ../input.txt &

echo "Run Worker"

../mrworker ../../apps/wordcount.so &
../mrworker ../../apps/wordcount.so &
../mrworker ../../apps/wordcount.so &

wait

sort mr-out* | grep . >answer
