username="Unous1996"
netid="aol3"
password="fakepassword"
git_password="fakepassword"

git_repo="https://github.com/username/cs425_sp19_mp2"

for i in {2..4};
do

if ((i < 10))
then

# sshpass -p ${password} ssh -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "hostname"

if sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "test -d ${git_repo}"
then

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "cd ${git_repo}; git checkout . ; git pull https://github.com/Unous1996/cs425_sp19_mp2.git"

echo ""

else

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-0${i}.cs.illinois.edu "cd /home/${netid}/mp2; git checkout . ;git pull https://github.com/Unous1996/cs425_sp19_mp2.git"

echo ""

fi

else

# sshpass -p ${password} ssh -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "hostname"

if sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "test -d ${git_repo}"

then

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "cd ${git_repo}; git checkout . ;git pull https://github.com/Unous1996/cs425_sp19_mp2.git"

echo ""

else

sshpass -p ${password} ssh -t -o "StrictHostKeyChecking no" ${netid}@sp19-cs425-g16-${i}.cs.illinois.edu "cd /home/${netid}/mp2; git checkout . ;git pull https://github.com/Unous1996/cs425_sp19_mp2.git"

echo ""

fi

fi

done
