// Package parsehelpers provides parse helpers for CLI applications.
// This package does not depend on any specific CLI library such as github.com/urfave/cli .
package parsehelpers

type EncArgs struct {
	GPGHomedir   string   // --gpg-homedir
	GPGVersion   string   // --gpg-version
	Key          []string // --key
	Recipient    []string // --recipient
	DecRecipient []string // --dec-recipient
}
