port=5000

for i in 0
do
	newport=`expr $port + $i`
	./main node $newport 
done


