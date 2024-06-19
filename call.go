package containerd

import "demo/pkg/cri/instrument"

func Call() {

	_ = new(instrument.InstrumentedService).Status

	_ = new(instrument.InstrumentedService).RunPodSandbox
}
