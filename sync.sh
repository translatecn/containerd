set -ex
rm -rf bin
\rm -rf ./log/*
bash build.sh
ssh -t root@vm rm -rf /tmp/*
rsync -aPc ./bin root@vm:/tmp
md5 ./bin/containerd
ssh -t root@vm systemctl disable containerd
ssh -t root@vm systemctl stop containerd
ssh -t root@vm cp -rf /tmp/bin/* /usr/bin/
ssh -t root@vm md5sum /usr/bin/containerd
ssh -t root@vm reboot
# rm -rf /tmp/* && cd /tmp && \rm -rf dbus* system* vmware* bin && systemctl start containerd



# tsz ./*