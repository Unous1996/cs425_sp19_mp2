read -p "Please enter your username:" username
read -p "Please enter your netid:" netid
read -sp "Please enter your password for your netid:" password
echo ""
read -sp "Please enter your github password:" git_password
echo ""

git_repo="https://github.com/username/cs425_sp19_mp2"

arr=(1 2 3 4 5 6 7 8 9 10)

for i in ${arr[*]};
do

if ((i < 10))
then

# sshpass -p ${password} ssh -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "hostname"

if sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "test -d ${git_repo}"
then

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "cd ${git_repo}; cd bandwidth; touch bandwidth${i}_5001.csv; touch bandwidth${i}_5002.csv; touch bandwidth${i}_5003.csv; touch bandwidth${i}_5004.csv; touch bandwidth${i}_5005.csv; touch bandwidth${i}_5006.csv; touch bandwidth${i}_5007.csv; touch bandwidth${i}_5008.csv; touch bandwidth${i}_5009.csv; touch bandwidth${i}_5010.csv; touch bandwidth${i}_5011.csv"

echo ""

else

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "cd /home/${netid}/mp2; cd bandwidth; touch bandwidth${i}_5001.csv; touch bandwidth${i}_5002.csv; touch bandwidth${i}_5003.csv; touch bandwidth${i}_5004.csv; touch bandwidth${i}_5005.csv; touch bandwidth${i}_5006.csv; touch bandwidth${i}_5007.csv; touch bandwidth${i}_5008.csv; touch bandwidth${i}_5009.csv; touch bandwidth${i}_5010.csv; touch bandwidth${i}_5011.csv"

echo ""

fi

else

# sshpass -p ${password} ssh -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "hostname"

if sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "test -d ${git_repo}"

then

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "cd ${git_repo}; cd bandwidth; touch bandwidth${i}_5001.csv; touch bandwidth${i}_5002.csv; touch bandwidth${i}_5003.csv; touch bandwidth${i}_5004.csv; touch bandwidth${i}_5005.csv; touch bandwidth${i}_5006.csv; touch bandwidth${i}_5007.csv; touch bandwidth${i}_5008.csv; touch bandwidth${i}_5009.csv; touch bandwidth${i}_5010.csv; touch bandwidth${i}_5011.csv"

echo ""

else

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "cd /home/${netid}/mp2;cd bandwidth; touch bandwidth${i}_5001.csv; touch bandwidth${i}_5002.csv; touch bandwidth${i}_5003.csv; touch bandwidth${i}_5004.csv; touch bandwidth${i}_5005.csv; touch bandwidth${i}_5006.csv; touch bandwidth${i}_5007.csv; touch bandwidth${i}_5008.csv; touch bandwidth${i}_5009.csv; touch bandwidth${i}_5010.csv; touch bandwidth${i}_5011.csv"

echo ""

fi

fi

done
