pkill -9 dlv
pkill -9 containerd_bin
pkill -9 ctr_bin


#rm -rf /var/lib/containers/*
#rm -rf /var/lib/containerd/*
#rm -rf /run/containerd/*
#rm -rf /etc/kubernetes/*
#rm -rf /etc/containers/*
#rm -rf /etc/cni/net.d/*
#rm -rf /var/lib/cni/*


mkdir -p /etc/cni/net.d
mkdir -p /run/flannel

cat > /run/flannel/subnet.env << EOF
FLANNEL_NETWORK=100.64.0.0/16
FLANNEL_SUBNET=100.64.0.1/24
FLANNEL_MTU=1450
FLANNEL_IPMASQ=true
EOF

cat > /etc/cni/net.d/10-flannel.conflist << EOF
{
  "cniVersion": "0.3.1",
  "name": "cbr0",
  "plugins": [
    {
      "type": "flannel",
      "delegate": {
        "hairpinMode": true,
        "isDefaultGateway": true
      }
    },
    {
      "type": "portmap",
      "capabilities": {
        "portMappings": true
      }
    }
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
go build -o containerd-shim-runc-v2 ./cmd/containerd-shim-runc-v2

# go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/internal/otlpconfig/envconfig.go
# err := bsp.e.ExportSpans(ctx, bsp.batch)
#export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://10.230.205.190:5080/api/default/traces
#export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
#export OTEL_EXPORTER_OTLP_TRACES_HEADERS=Authorization='Basic cm9vdEBleGFtcGxlLmNvbTpPbmRxZE9BSFZLc3lKTWVT'
export DEBUG=1

# docker run -d \
#      --name openobserve \
#      -v $PWD/data:/data \
#      -p 5080:5080 -p 5081:5081 \
#      -e ZO_ROOT_USER_EMAIL="root@example.com" \
#      -e ZO_ROOT_USER_PASSWORD="Complexpass#123" \
#      public.ecr.aws/zinclabs/openobserve:latest

cp -rf ctr_bin /usr/local/bin/ctr
dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./containerd_bin -- -c ./cmd/containerd/config.toml
#dlv --listen=:12345 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- i export nginx.img docker.m.daocloud.io/library/nginx:alpine
#dlv --listen=:12345 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- i mount docker.m.daocloud.io/library/nginx:alpine /123
#dlv --listen=:12345 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- i pull docker.m.daocloud.io/library/nginx:alpine &
#dlv --listen=:12346 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- i container create -t docker.m.daocloud.io/library/nginx:alpine nginx_1 sh &
#dlv --listen=:12347 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- task start nginx_1
#dlv --listen=:12345 --headless=true --api-version=2 --accept-multiclient exec ./ctr_bin -- task ls
# /Users/acejilam/Desktop/containerd/ctr_bin i pull docker.m.daocloud.io/library/nginx:alpine
#
#go install github.com/go-delve/delve/cmd/dlv@master
#go install github.com/trzsz/trzsz-go/cmd/...@latest

#ctr container create --sandbox docker.m.daocloud.io/library/nginx:alpine 69ffb43cae82f2bd4b2d367106d5b8ab33644a12ea1b6605b6e9468f3608107a

#/Users/acejilam/Desktop/containerd/ctr_bin i pull registry.cn-hangzhou.aliyuncs.com/google_containers/pause:3.7
#/Users/acejilam/Desktop/containerd/ctr_bin containers create --config /Users/acejilam/Desktop/containerd/cmd/ctr/commands/containers/examples/pause-base64.json 4ca9820f3b578114c455aa2b454b368f9e71d64f6c3e9b380ff326a80d46ee91


#/Users/acejilam/Desktop/containerd/ctr_bin i pull docker.m.daocloud.io/library/nginx:alpine
#/Users/acejilam/Desktop/containerd/ctr_bin container create -t docker.m.daocloud.io/library/nginx:alpine nginx_1 sh
#/Users/acejilam/Desktop/containerd/ctr_bin task start nginx_1

