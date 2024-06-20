# 运行 pod 沙箱
sanbox_id=$(./bin/crictl runp --runtime=runc ./examples/sandbox-config.json)
./bin/crictl inspectp $sanbox_id
./bin/crictl pods

./bin/crictl pull registry.cn-hangzhou.aliyuncs.com/acejilam/busybox

# 在 pod 沙箱中创建容器
container_id=$(./bin/crictl create $sanbox_id ./examples/container-config.json ./examples/sandbox-config.json)
./bin/crictl start $container_id
./bin/crictl ps

./bin/crictl exec -i -t $container_id ls


# https://zhuanlan.zhihu.com/p/431406216
# https://zhuanlan.zhihu.com/p/360153886