./bin/crictl runp --runtime=runc ./examples/sandbox-config.json
./bin/crictl inspectp 53675eb8893ee865ccdf2b19caee7f9883a2583b8622fea0f79effb04e73665e

# 创建容器，传递先前创建的 Pod 的 ID、容器配置文件和 Pod 配置文件
./bin/crictl create 96de16c03df15a10c007bec284b98dd6dc9e34c0852edfd7f96deeadc845e20f ./examples/container-config.json ./examples/sandbox-config.json
./bin/crictl start 3e025dd50a72d956c4f14881fbb5b1080c9275674e95fb67f965f6478a957d60
./bin/crictl ps



https://zhuanlan.zhihu.com/p/431406216
https://zhuanlan.zhihu.com/p/360153886