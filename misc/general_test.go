package misc

import "testing"

func TestContains(t *testing.T) {
	s := []string{"one", "two", "three", "four"}
	if Contains(s, "no") {
		t.Error("Shouldn't contain")
	}
	if Contains(s, "") {
		t.Error("Shouldn't contain")
	}
	if !Contains(s, "one") {
		t.Error("Should contain")
	}
	if !Contains(s, "four") {
		t.Error("Should contain")
	}
}
