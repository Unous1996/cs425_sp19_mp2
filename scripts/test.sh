cd ..
go build main.go

cd scripts/

sh buildkill.sh
pssh -t 10000 -i -h ../hosts.txt "cd /home/aol3/mp2/;sh simulate.sh $(date +%s)"
