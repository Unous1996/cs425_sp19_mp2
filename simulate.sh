port=5000
mkdir -p logs/"$(date "+%F-%T")"

for i in 1
do
	newport=`expr $port + $i`
	./main node $newport $(date "+%F-%T") &
done


