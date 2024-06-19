package platforms

// DefaultString returns the default string specifier for the platform.
func DefaultString() string {
	return Format(DefaultSpec())
}

// DefaultStrict returns strict form of Default.
func DefaultStrict() MatchComparer {
	return OnlyStrict(DefaultSpec())
}
