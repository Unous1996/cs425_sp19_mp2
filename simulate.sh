port=5000
d=`date +%m-%d-%H-%M`
mkdir -p logs/"$d/latency"
mkdir -p logs/"$d/bandwidth"

for i in {0..11}
do
	newport=`expr $port + $i`
	./main node $newport $d &
done
