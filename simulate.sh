port=5000

if [ "$1" != "" ]; then
    echo "Positional parameter 1 contains something"
else
    echo "Positional parameter 1 is empty"
fi

mkdir -p logs/"$d/latency"
mkdir -p logs/"$d/bandwidth"

for i in {0..10}
do
	newport=`expr $port + $i`
	./main node $newport "$1" &
done
