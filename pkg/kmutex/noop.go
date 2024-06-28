package kmutex

import "context"

func NewNoop() KeyedLocker {
	return &noopMutex{}
}

type noopMutex struct {
}

func (*noopMutex) Lock(_ context.Context, _ string) error {
	return nil
}

func (*noopMutex) Unlock(_ string) {
}
