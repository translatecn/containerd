package stdio

import (
	"context"
	"sync"

	"demo/pkg/console"
)

// Platform handles platform-specific behavior that may differs across
// platform implementations
type Platform interface {
	CopyConsole(ctx context.Context, console console.Console, id, stdin, stdout, stderr string,
		wg *sync.WaitGroup) (console.Console, error)
	ShutdownConsole(ctx context.Context, console console.Console) error
	Close() error
}
