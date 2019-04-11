cd ..
go build main.go
git commit -a -m "$(date +%s)"
git push origin master
cd scripts/
sh buildkill.sh
sh git_deploy.sh
sh buildkill.sh
pssh -t 10000 -i -h hosts.txt "cd /home/aol3/mp2/;sh simulate.sh $(date +%s)"