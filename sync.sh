ssh -t root@vm rm -rf /Users/acejilam/Desktop/containerd
ssh -t root@vm mkdir -p /Users/acejilam/Desktop/containerd
rsync -aPc . root@vm:/Users/acejilam/Desktop/containerd
ssh -t root@vm rm -rf /Users/acejilam/Desktop/containerd/.idea
ssh -t root@vm bash /Users/acejilam/Desktop/containerd/debug.sh