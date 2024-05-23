systemd─┬─ModemManager───2*[{ModemManager}]
        ├─containerd-shim─┬─pause
        │                 ├─tini───calico-typha───12*[{calico-typha}]
        │                 └─11*[{containerd-shim}]



[root@vm manifests]# pstree -p 4224
containerd-shim(4224)─┬─pause(4269)
                      ├─tini(4450)───calico-typha(4463)─┬─{calico-typha}(4464)
                      │                                 ├─{calico-typha}(4465)
                      │                                 ├─{calico-typha}(4466)
                      │                                 ├─{calico-typha}(4467)
                      │                                 ├─{calico-typha}(4468)
                      │                                 ├─{calico-typha}(4469)
                      │                                 ├─{calico-typha}(4470)
                      │                                 ├─{calico-typha}(4471)
                      │                                 ├─{calico-typha}(4472)
                      │                                 ├─{calico-typha}(4473)
                      │                                 ├─{calico-typha}(4474)
                      │                                 └─{calico-typha}(4475)
                      ├─{containerd-shim}(4225)
                      ├─{containerd-shim}(4226)
                      ├─{containerd-shim}(4227)
                      ├─{containerd-shim}(4228)
                      ├─{containerd-shim}(4229)
                      ├─{containerd-shim}(4230)
                      ├─{containerd-shim}(4231)
                      ├─{containerd-shim}(4232)
                      ├─{containerd-shim}(4233)
                      ├─{containerd-shim}(4456)
                      └─{containerd-shim}(4476)

