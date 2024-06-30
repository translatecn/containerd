package api

// ToOCI returns an OCI Env entry for the KeyValue.
func (e *KeyValue) ToOCI() string {
	return e.Key + "=" + e.Value
}

// FromOCIEnv returns KeyValues from an OCI runtime Spec environment.

// IsMarkedForRemoval checks if an environment variable is marked for removal.
func (e *KeyValue) IsMarkedForRemoval() (string, bool) {
	key, marked := IsMarkedForRemoval(e.Key)
	return key, marked
}
