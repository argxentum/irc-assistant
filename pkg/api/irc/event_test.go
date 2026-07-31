package irc

import "testing"

func TestRecipientWithNoArguments(t *testing.T) {
	e := &Event{}

	recipient, entityType := e.Recipient()
	if recipient != "" {
		t.Fatalf("recipient = %q, want empty", recipient)
	}
	if entityType != EntityTypeUser {
		t.Fatalf("entity type = %q, want %q", entityType, EntityTypeUser)
	}

	// Labels calls Recipient too; malformed server input must remain safe while
	// diagnostic labels are being created.
	_ = e.Labels()
}

func TestParseMaskRejectsEmptyInput(t *testing.T) {
	if mask := ParseMask(""); mask != nil {
		t.Fatalf("ParseMask returned %#v for empty input", mask)
	}
}

func TestMaskMatchesHandlesNil(t *testing.T) {
	mask := ParseMask("nick!user@example.com")
	if mask.Matches(nil) {
		t.Fatal("mask unexpectedly matched nil")
	}

	var nilMask *Mask
	if nilMask.Matches(mask) {
		t.Fatal("nil mask unexpectedly matched")
	}
}
