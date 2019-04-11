read -p "Please enter your username:" username
read -p "Please enter your netid:" netid
read -sp "Please enter your password for your netid:(whatever)" password
echo ""
read -sp "Please enter your github password:(whatever)" git_password
echo ""

git_repo="https://github.com/username/cs425_sp19_mp2"
# arr=(1 2 3 4 5 6 7 8 9 10)
arr=(2 3 4 5 6)
# arr=(2 3 4 5 6 7 8 9 10)
# arr=(2 3)

for vmnumber in ${arr[*]};
do

if ((vmnumber < 10))
then

#sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${vmnumber}.cs.illinois.edu "cd /home/${netid}/mp2/latency; cat latency${vmnumber}_5000.csv  latency${vmnumber}_5001.csv latency${vmnumber}_5002.csv latency${vmnumber}_5003.csv  latency${vmnumber}_5004.csv latency${vmnumber}_5005.csv latency${vmnumber}_5006.csv  latency${vmnumber}_5007.csv latency${vmnumber}_5008.csv latency${vmnumber}_5009.csv  latency${vmnumber}_5010.csv latency${vmnumber}_5011.csv > latency_${vmnumber}_overall.csv; cd ../bandwidth; cat bandwidth${vmnumber}_5000.csv  bandwidth${vmnumber}_5001.csv bandwidth${vmnumber}_5002.csv bandwidth${vmnumber}_5003.csv  bandwidth${vmnumber}_5004.csv bandwidth${vmnumber}_5005.csv bandwidth${vmnumber}_5006.csv  bandwidth${vmnumber}_5007.csv bandwidth${vmnumber}_5008.csv bandwidth${vmnumber}_5009.csv  bandwidth${vmnumber}_5010.csv bandwidth${vmnumber}_5011.csv > bandwidth_${vmnumber}_overall.csv;"
sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${vmnumber}.cs.illinois.edu "cd /home/${netid}/mp2/latency; cat * > latency_${vmnumber}_overall.csv; cd ../bandwidth; cat * > bandwidth_${vmnumber}_overall.csv; cd ../balance; cat * > balance_${vmnumber}_overall.csv; cd ../blocklatency; cat * > blocklatency_${vmnumber}_overall.csv; cd ../commitlatency; cat * > commitlatency_${vmnumber}_overall.csv; cd ../split; cat * > split_${vmnumber}_overall.csv"

else

#sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${vmnumber}.cs.illinois.edu "cd /home/${netid}/mp2/latency; cat latency${vmnumber}_5000.csv  latency${vmnumber}_5001.csv latency${vmnumber}_5002.csv latency${vmnumber}_5003.csv  latency${vmnumber}_5004.csv latency${vmnumber}_5005.csv latency${vmnumber}_5006.csv  latency${vmnumber}_5007.csv latency${vmnumber}_5008.csv latency${vmnumber}_5009.csv  latency${vmnumber}_5010.csv latency${vmnumber}_5011.csv > latency_${vmnumber}_overall.csv; cd ../bandwidth; cat bandwidth${vmnumber}_5000.csv  bandwidth${vmnumber}_5001.csv bandwidth${vmnumber}_5002.csv bandwidth${vmnumber}_5003.csv  bandwidth${vmnumber}_5004.csv bandwidth${vmnumber}_5005.csv bandwidth${vmnumber}_5006.csv  bandwidth${vmnumber}_5007.csv bandwidth${vmnumber}_5008.csv bandwidth${vmnumber}_5009.csv  bandwidth${vmnumber}_5010.csv bandwidth${vmnumber}_5011.csv > bandwidth_${vmnumber}_overall.csv;"
sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${vmnumber}.cs.illinois.edu "cd /home/${netid}/mp2/latency; cat * > latency_${vmnumber}_overall.csv; cd ../bandwidth; cat * > bandwidth_${vmnumber}_overall.csv; cd ../balance; cat * > balance_${vmnumber}_overall.csv; cd ../blocklatency; cat * > blocklatency_${vmnumber}_overall.csv; cd ../commitlatency; cat * > commitlatency_${vmnumber}_overall.csv; cd ../split; cat * > split_${vmnumber}_overall.csv"

fi

done

foldername=$(date +"%m-%d-%H-%M")
echo $foldername

mkdir -p ../results/${foldername}/latency
mkdir -p ../results/${foldername}/bandwidth
mkdir -p ../results/${foldername}/balance
mkdir -p ../results/${foldername}/blocklatency
mkdir -p ../results/${foldername}/commitlatency
mkdir -p ../results/${foldername}/split

for vmnumber in ${arr[*]};
do
	if((vmnumber < 10))
	then
		scp ${netid}@sp19-cs425-g16-0${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/latency/latency_${vmnumber}_overall.csv ../results/${foldername}/latency
		scp ${netid}@sp19-cs425-g16-0${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/bandwidth/bandwidth_${vmnumber}_overall.csv ../results/${foldername}/bandwidth
		scp ${netid}@sp19-cs425-g16-0${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/balance/balance_${vmnumber}_overall.csv ../results/${foldername}/balance	
		scp ${netid}@sp19-cs425-g16-0${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/blocklatency/blocklatency_${vmnumber}_overall.csv ../results/${foldername}/blocklatency
		scp ${netid}@sp19-cs425-g16-0${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/commitlatency/commitlatency_${vmnumber}_overall.csv ../results/${foldername}/commitlatency
		scp ${netid}@sp19-cs425-g16-0${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/split/split_${vmnumber}_overall.csv ../results/${foldername}/split	
	else
		scp ${netid}@sp19-cs425-g16-${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/latency/latency_${vmnumber}_overall.csv ../results/${foldername}/latency
		scp ${netid}@sp19-cs425-g16-${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/bandwidth/bandwidth_${vmnumber}_overall.csv ../results/${foldername}/bandwidth
		scp ${netid}@sp19-cs425-g16-${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/balance/balance_${vmnumber}_overall.csv ../results/${foldername}/balance	
		scp ${netid}@sp19-cs425-g16-${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/blocklatency/blocklatency_${vmnumber}_overall.csv ../results/${foldername}/blocklatency
		scp ${netid}@sp19-cs425-g16-${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/commitlatency/commitlatency_${vmnumber}_overall.csv ../results/${foldername}/commitlatency
		scp ${netid}@sp19-cs425-g16-${vmnumber}.cs.illinois.edu:/home/${netid}/mp2/split/split_${vmnumber}_overall.csv ../results/${foldername}/split	
	fi
done

cd ../results/${foldername}/latency
cat * > latency_${foldername}.csv
cd ../bandwidth
cat * > bandwidth_${foldername}.csv
cd ../balance
cat * > balance_${foldername}.csv
cd ../blocklatency
cat * > blocklatency_${foldername}.csv
cd ../commitlatency
cat * > commitlatency_${foldername}.csv
cd ../split
cat * > split_${foldername}.csv

subl ..



