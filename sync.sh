ssh -t root@172.16.244.147 rm -rf /Users/acejilam/Desktop/containerd
ssh -t root@172.16.244.147 mkdir -p /Users/acejilam/Desktop/containerd
rsync -aPc . root@172.16.244.147:/Users/acejilam/Desktop/containerd
ssh -t root@172.16.244.147 bash /Users/acejilam/Desktop/containerd/debug.sh