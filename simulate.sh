port=5000

for i in {1..6}
do
	newport=`expr $port + $i`
	./main node $newport &
done


