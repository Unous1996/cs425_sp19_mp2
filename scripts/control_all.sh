read -sp "Please enter your neitd:" netid
read -sp "Please enter your password:" password

echo ""

for i in {1..10}
do

if ((i < 10))
then

sshpass -p ${password} ssh -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "hostname"

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "$1"

echo ""

else

sshpass -p ${password} ssh -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "hostname"

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "$1"

echo ""

fi

done
