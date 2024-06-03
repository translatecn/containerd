pkill -9 dlv
pkill -9 containerd_bin
pkill -9 ctr_bin


rm -rf /var/lib/containers/*
rm -rf /var/lib/containerd/*
rm -rf /run/containerd/*
rm -rf /etc/containers/*
rm -rf /etc/cni/net.d/*


mkdir -p /etc/cni/net.d
cat > /etc/cni/net.d/10-calico.conflist << EOF
{
  "name": "k8s-pod-network",
  "cniVersion": "0.3.1",
  "plugins": [
  ]
}
EOF

# shellcheck disable=SC2164
cd /Users/acejilam/Desktop/containerd
source /etc/profile

rm -rf containerd_bin ctr_bin || echo success

# shellcheck disable=SC2046
go build -o containerd_bin ./cmd/containerd/main.go
go build -o ctr_bin ./cmd/ctr/main.go

dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./containerd_bin -- -c ./cmd/containerd/config.toml &

#/Users/acejilam/Desktop/containerd/ctr_bin i pull registry.cn-hangzhou.aliyuncs.com/acejilam/x:v1
#dlv --listen=:12345 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- i export nginx.img docker.io/library/nginx:alpine
#dlv --listen=:12345 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- i mount docker.io/library/nginx:alpine /123
#dlv --listen=:12345 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- i pull docker.io/library/nginx:alpine
dlv --listen=:12345 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- container create --sandbox docker.io/library/nginx:alpine 69ffb43cae82f2bd4b2d367106d5b8ab33644a12ea1b6605b6e9468f3608107a

# /Users/acejilam/Desktop/containerd/ctr_bin i pull docker.io/library/nginx:alpine
#
#go install github.com/go-delve/delve/cmd/dlv@master
#go install github.com/trzsz/trzsz-go/cmd/...@latest







#ctr container create --sandbox docker.io/library/nginx:alpine 69ffb43cae82f2bd4b2d367106d5b8ab33644a12ea1b6605b6e9468f3608107a











