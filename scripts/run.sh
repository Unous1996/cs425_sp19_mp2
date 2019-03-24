cd ..
git commit -a -m "$(date)"
git push origin master
cd scripts/
sh git_deploy.sh
sh buildkill.sh
pssh -i -h ~/.pssh_hosts_files "cd /home/aol3/mp2/;sh simulate.sh $(date)"
