read -p "Please enter your netid:" netid

sshpass -p "my_password_here" ssh ${netid}@sp19-cs425-g16-02.cs.illinois.edu /home/${netid}/mp2/simulate.sh