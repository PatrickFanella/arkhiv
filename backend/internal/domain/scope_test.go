package domain

import "testing"

func TestExplicitCollectionScope(t *testing.T) {
	for _, input := range [][2]string{{"one", "two"}, {"", "one,one"}, {"", "one,"}, {"", "../one"}, {"bad/one", ""}} {
		if _, err := ParseCollectionScope(input[0], input[1]); err == nil {
			t.Fatal("invalid scope accepted", input)
		}
	}
	s, err := ParseCollectionScope("", "one, two")
	if err != nil || !s.Allows("one") || !s.Allows("two") || s.Allows("foreign") || s.Default() != "" {
		t.Fatal("multi collection boundary", s, err)
	}
	legacy, err := ParseCollectionScope("one", "")
	if err != nil || legacy.Default() != "one" || legacy.Allows("two") {
		t.Fatal("legacy collection boundary", legacy, err)
	}
	if (CollectionScope{}).Allows("one") || (CollectionScope{}).Default() != "" {
		t.Fatal("explicit empty scope widened")
	}
	if !CollectionScope(nil).Allows("one") || CollectionScope(nil).Allows("bad/one") {
		t.Fatal("unrestricted scope accepts invalid identity")
	}
}
