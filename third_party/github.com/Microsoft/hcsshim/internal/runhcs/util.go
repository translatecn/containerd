package runhcs

const (
	SafePipePrefix = `\\.\pipe\ProtectedPrefix\Administrators\`
)

// ShimSuccess is the byte stream returned on a successful operation.
var ShimSuccess = []byte{0, 'O', 'K', 0}
