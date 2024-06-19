package mergemaps

// Merge recursively merges map `fromMap` into map `ToMap`. Any pre-existing values
// in ToMap are overwritten. Values in fromMap are added to ToMap.
// From http://stackoverflow.com/questions/40491438/merging-two-json-strings-in-golang

// MergeJSON merges the contents of a JSON string into an object representation,
// returning a new object suitable for translating to JSON.
