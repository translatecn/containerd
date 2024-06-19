package docker

import (
	"sort"
)

// Sort sorts string references preferring higher information references
// The precedence is as follows:
// 1. Name + Tag + Digest
// 2. Name + Tag
// 3. Name + Digest
// 4. Name
// 5. Digest
// 6. Parse error
func Sort(references []string) []string {
	var prefs []Reference
	var bad []string

	for _, ref := range references {
		pref, err := ParseAnyReference(ref)
		if err != nil {
			bad = append(bad, ref)
		} else {
			prefs = append(prefs, pref)
		}
	}
	sort.Slice(prefs, func(a, b int) bool {
		ar := refRank(prefs[a])
		br := refRank(prefs[b])
		if ar == br {
			return prefs[a].String() < prefs[b].String()
		}
		return ar < br
	})
	sort.Strings(bad)
	var refs []string
	for _, pref := range prefs {
		refs = append(refs, pref.String())
	}
	return append(refs, bad...)
}

func refRank(ref Reference) uint8 {
	if _, ok := ref.(Named); ok {
		if _, ok = ref.(Tagged); ok {
			if _, ok = ref.(Digested); ok {
				return 1
			}
			return 2
		}
		if _, ok = ref.(Digested); ok {
			return 3
		}
		return 4
	}
	return 5
}
