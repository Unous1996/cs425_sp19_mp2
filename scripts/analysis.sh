cd ../latency
arr=(0 1 2 3)
for vmnumber in ${arr[*]}
do 
	cat "latency${vmnumber}_5000".csv  "latency${vmnumber}_5001".csv "latency${vmnumber}_5002".csv > "latency_$vmnumber_overall".csv 
done
