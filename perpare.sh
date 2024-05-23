docker run -it centos:7 bash
yum install wget net-tools -y
wget -O /etc/yum.repos.d/docker-ce.repo https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo
yum list | grep containerd
yum -y install containerd.io
rpm -qa | grep containerd
ctr version
ls /usr/bin/

containerd config default > /etc/containerd/config.toml