port=5000

for i in {1..100}
do
	newport=`expr $port + $i`
	./main node $newport &
done


