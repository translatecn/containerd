- https://zhuanlan.zhihu.com/p/422522890
- https://zhuanlan.zhihu.com/p/676356164
- https://www.jianshu.com/p/712273fd1cfd
- https://www.jianshu.com/p/87b4876fbf65
- https://www.jianshu.com/p/d446121dbfc2
- https://blog.csdn.net/power886/article/details/137969087



- https://github.com/containerd/nerdctl
- https://github.com/kubernetes-sigs/cri-tools
- https://github.com/opencontainers/runc
- https://github.com/containernetworking/plugins

- https://blog.csdn.net/andlee/article/details/134768876

```
├── io.containerd.content.v1.content
│   └── ingest
│       └── 641c94350954f6c744abc0657b67ef4f5e74db6a9998b7797876be6ee2177111
│           ├── data
│           ├── ref
│           ├── startedat
│           ├── total
│           └── updatedat


- v1
  - default
    - content
      - ingests
        - index-sha256:be65f488b7764ad3638f236b7b515b3678369a5124c47b8d32916d
          ref: default/1/index-sha256:be65f488b7764ad3638f236b7b515b3678369a5

```


- /var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/metadata.db



- alias runc='runc --root /run/containerd/runc/default --log /run/containerd/io.containerd.runtime.v2.task/default/nginx_1/log.json --log-format json'
- runc create --bundle /run/containerd/io.containerd.runtime.v2.task/default/nginx_1 --pid-file /run/containerd/io.containerd.runtime.v2.task/default/nginx_1/init.pid --console-socket /run/user/0/pty1877964415/pty.sock nginx_1
- runc start nginx_1
- runc ps --format json nginx_1
- runc pause nginx_1
- runc delete nginx_1
- runc exec --process /run/user/0/runc-process135976610 --console-socket /run/user/0/pty2154878890/pty.sock --detach --pid-file /run/containerd/io.containerd.runtime.v2.task/default/nginx_1/2512.pid nginx_1
- runc kill nginx_1 3


```
wget https://github.com/containernetworking/plugins/releases/download/v1.1.0/cni-plugins-linux-amd64-v1.1.0.tgz
mkdir -p /opt/cni/bin
tar xvzf cni-plugins-linux-amd64-v1.1.0.tgz -C /opt/cni/bin


mkdir -p /tmp/etc/cni/net.d/
cat << EOF | tee /tmp/etc/cni/net.d//10-containerd-net.conflist
{
  "cniVersion": "0.4.0",
  "name": "containerd-net",
  "plugins": [
    {
      "type": "bridge",
      "bridge": "cni0",
      "isGateway": true,
      "ipMasq": true,
      "promiscMode": true,
      "ipam": {
        "type": "host-local",
        "ranges": [
          [{
            "subnet": "10.88.0.0/16"
          }],
          [{
            "subnet": "2001:4860:4860::/64"
          }]
        ],
        "routes": [
          { "dst": "0.0.0.0/0" },
          { "dst": "::/0" }
        ]
      }
    },
    {
      "type": "portmap",
      "capabilities": {"portMappings": true}
    }
  ]
}



```


cri-tools
crictl