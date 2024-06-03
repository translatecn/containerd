set -ex

#docker run -it centos:7 bash
#curl -o /etc/yum.repos.d/CentOS-Base.repo http://mirrors.aliyun.com/repo/Centos-8.repo
#yum clean all
#yum makecache
#yum install wget net-tools -y
#wget -O /etc/yum.repos.d/docker-ce.repo https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo
#
#yum list | grep containerd
#yum -y install containerd.io
#rpm -qa | grep containerd
#ctr version
#ls /usr/bin/
#
#containerd config default > /etc/containerd/config.toml
#systemctl restart containerd

#yum autoremove containerd.io -y

#ctr i pull registry.cn-hangzhou.aliyuncs.com/acejilam/x:v1

cat > /usr/lib/systemd/system/containerd.service <<EOF
[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target local-fs.target

[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/usr/local/bin/containerd

Type=notify
Delegate=yes
KillMode=process
Restart=always
RestartSec=5

# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNPROC=infinity
LimitCORE=infinity

# Comment TasksMax if your systemd version does not supports it.
# Only systemd 226 and above support this version.
TasksMax=infinity
OOMScoreAdjust=-999

[Install]
WantedBy=multi-user.target
EOF

export https_proxy=http://172.20.10.248:1080
systemctl stop containerd
systemctl disable containerd
Version="1.7.17"
Arch="amd64"
curl -LO https://github.com/containerd/containerd/releases/download/v${Version}/containerd-${Version}-linux-${Arch}.tar.gz
unset https_proxy

tar -xvf containerd-${Version}-linux-${Arch}.tar.gz

\cp -rf ./bin/* /usr/local/bin/
mkdir /etc/containerd
source /etc/profile
containerd config default > /etc/containerd/config.toml

systemctl restart containerd
systemctl status containerd




rm -rf /usr/local/bin/containerd
rm -rf /usr/local/bin/containerd-shim
rm -rf /usr/local/bin/containerd-shim-runc-v1
rm -rf /usr/local/bin/containerd-shim-runc-v2
rm -rf /usr/local/bin/containerd-stress
rm -rf /usr/local/bin/ctr
systemctl stop containerd
systemctl disable containerd