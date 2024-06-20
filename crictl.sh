# 运行 pod 沙箱  RunPodSandbox
sanbox_id=$(./bin/crictl runp --runtime=runc ./examples/sandbox-config.yaml)
./bin/crictl inspectp $sanbox_id
./bin/crictl pods

./bin/crictl pull registry.cn-hangzhou.aliyuncs.com/acejilam/busybox

# 在 pod 沙箱中创建容器
container_id=$(./bin/crictl create $sanbox_id ./examples/container-config.yaml ./examples/sandbox-config.yaml)
./bin/crictl start $container_id
./bin/crictl ps

./bin/crictl exec -i -t $container_id ls


# https://zhuanlan.zhihu.com/p/431406216
# https://zhuanlan.zhihu.com/p/360153886

# ["GOMAXPROCS=2","GRPC_ADDRESS=/run/containerd/containerd.sock","MAX_SHIM_VERSION=2","NAMESPACE=k8s.io","TTRPC_ADDRESS=/run/containerd/containerd.sock.ttrpc"]

# 创建 /run/containerd/s/36115640fe64cfee54d4b98e7489129a8456524ca701effd0f8fa3993810af96 并打印
# /Users/acejilam/Desktop/containerd/containerd-shim-runc-v2 -namespace k8s.io -address /run/containerd/containerd.sock -publish-binary /Users/acejilam/Desktop/con/Users/acejilam/Desktop/containerd/containerd-shim-runc-v2 -- -namespace k8s.io -address /run/containerd/containerd.sock -publish-binary /Users/acejilam/Desktop/ctainerd/containerd_bin -id f56fc531a7713ebd6a0ecea8024a55e895094f7138cc2344b0fc341ddb43b6cf start

# 传递 /run/containerd/s/36115640fe64cfee54d4b98e7489129a8456524ca701effd0f8fa3993810af96
# /Users/acejilam/Desktop/containerd/containerd-shim-runc-v2 -namespace k8s.io -address /run/containerd/containerd.sock -id f56fc531a7713ebd6a0ecea8024a55e895094f7138cc2344b0fc341ddb43b6cf
