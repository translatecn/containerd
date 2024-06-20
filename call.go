package containerd

import (
	pb "demo/over/api/cri/v1"
	"demo/pkg/cri/sbserver"
)

func main() {

	_ = new(sbserver.CriService).Version
	_ = new(sbserver.CriService).Status

	// cri接口接收的参数
	_ = pb.PodSandboxConfig{}
}
