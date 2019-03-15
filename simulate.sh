port=5000
mkdir -p logs/"$(date "+%F-%T")/latency"
mkdir -p logs/"$(date "+%F-%T")/bandwidth"
mkdir -p mergeresults/$(date "+%F-%T")

for i in {0..9}
do
	newport=`expr $port + $i`
	./main node $newport $(date "+%F-%T") &
done

cat logs/"$(date "+%F-%T")"/latency/* >> mergeresults/"$(date "+%F-%T")latency$(date "+%F-%T")".csv
cat logs/"$(date "+%F-%T")"/bandwidth/* >> mergeresults/"$(date "+%F-%T")bandwidth$(date "+%F-%T")".csv

