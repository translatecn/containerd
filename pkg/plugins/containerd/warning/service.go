package warning

import (
	"context"
	"demo/pkg/deprecation"
	"demo/pkg/log"
	"demo/pkg/plugin"
	"sync"
	"time"
)

type Service interface {
	Emit(context.Context, deprecation.Warning)
	Warnings() []Warning
}

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.WarningPlugin,
		ID:   plugin.DeprecationsPlugin,
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			return &service{warnings: make(map[deprecation.Warning]time.Time)}, nil
		},
	})
}

type Warning struct {
	ID             deprecation.Warning
	LastOccurrence time.Time
	Message        string
}

var _ Service = (*service)(nil)

type service struct {
	warnings map[deprecation.Warning]time.Time
	m        sync.RWMutex
}

func (s *service) Emit(ctx context.Context, warning deprecation.Warning) {
	if !deprecation.Valid(warning) {
		log.G(ctx).WithField("warningID", string(warning)).Warn("invalid deprecation warning")
		return
	}
	s.m.Lock()
	defer s.m.Unlock()
	s.warnings[warning] = time.Now()
}
func (s *service) Warnings() []Warning { // ctr_bin plugin ls
	s.m.RLock()
	defer s.m.RUnlock()
	var warnings []Warning
	for k, v := range s.warnings {
		msg, ok := deprecation.Message(k)
		if !ok {
			continue
		}
		warnings = append(warnings, Warning{
			ID:             k,
			LastOccurrence: v,
			Message:        msg,
		})
	}
	return warnings
}
