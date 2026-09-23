package ldapserver

import (
	"encoding/hex"
	"testing"

	ldap "newldap/internal/goldapmessage"
)

func TestBindResponseWire(t *testing.T) {
	res := NewBindResponse(0)
	m := ldap.NewLDAPMessageWithProtocolOp(res)
	m.SetMessageID(1)
	b, err := m.Write()
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	got := hex.EncodeToString(b.Bytes())
	want := "300c02010161070a010004000400"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestMessageID128Wire(t *testing.T) {
	res := NewBindResponse(0)
	m := ldap.NewLDAPMessageWithProtocolOp(res)
	m.SetMessageID(128)
	b, err := m.Write()
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	got := hex.EncodeToString(b.Bytes())
	// messageID 128 → 02 02 00 80；信封长度 13 (0x0d)
	want := "300d0202008061070a010004000400"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}
