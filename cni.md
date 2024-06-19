CNI_COMMAND=ADD param_1=value1 ./bridge < config.json


- CNI_COMMAND   :ADD\DEL\CHECK\VERSION
- CNI_CONTAINERD
- CNI_NETNS
- CNI_IFNAME
- CNI_ARGS
- CNI_PATH


```
# 创建网络
CONTAINERDID=$(nerdctl run -d --network none nginx)
PID=$(nerdctl inspect $CONTAINERDID -f '{{.State.Pid }}')
NET_NS_PATH=/proc/$PID/ns/net
# 方式1
nsenter $PID -n ip a
# 方式2
<!-- mount --bind /proc/$PID/ns/net /var/run/netns/zjz -->
<!-- ip netns exec zjz ip a  -->


{
    "cniVersion": "1.0.0",
    "name": "dbnet",
    "type": "bridge",
    "bridge": "mycni0",
    "isGateway": true,
    "keyA": [
        "some more",
        "plugin specific",
        "configuration"
    ],
    "ipam": {
        "type": "host-local",
        "subnet": "10.1.0.0/16",
        "routes": {
            "dst": "0.0.0.0/0"
        },
        "dns": {
            "nameservers": [
                "10.1.0.1"
            ]
        }
    }
}

# bridge 插件会调用 ipam 分配ip
CNI_COMMAND=ADD CNI_CONTAINERD=$CONTAINERDID CNI_NETNS=$NET_NS_PATH
CNI_IFNAME=eht0 CNI_PATH=/opt/cni/bin /opt/cni/bin/bridge < ~/bridge.json


nsenter $PID -n ip a
nsenter $PID -n ip route
nsenter $PID -n curl ....



CNI_COMMAND=ADD CNI_CONTAINERID=$CONTAINERID CNI_NETNS=$NET_NS_PAT HCNI_
IFNAME=etho CNI_PATH=/opt/eni/bin /opt/cni/bin/tuning < ~/tuning.json
```