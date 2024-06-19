package containerd

import "demo/pkg/cri/instrument"

func main() {

	_ = new(instrument.InstrumentedService).Version
	_ = new(instrument.InstrumentedService).Status
}
