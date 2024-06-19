package oc

// todo: break import cycle with "internal/hcs/errors.go" and reference errors defined there
// todo: add errors defined in "internal/guest/gcserror" (Hresult does not implement error)

// isAny returns true if errors.Is is true for any of the provided errors, errs.
