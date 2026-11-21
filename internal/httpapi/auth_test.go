package httpapi

import (
	"encoding/base64"
	"testing"
)

func TestPrincipalOwnerIDSeparatesHumanSubjectFromClientIdentity(t *testing.T) {
	principal := Principal{
		Issuer:   "https://issuer.example/auth/v1",
		Subject:  "user-123",
		ClientID: "crm-web-client",
	}
	if got, want := principal.OwnerID(), "user:"+base64.RawURLEncoding.EncodeToString([]byte("https://issuer.example/auth/v1"))+":"+base64.RawURLEncoding.EncodeToString([]byte("user-123")); got != want {
		t.Fatalf("OwnerID()=%q, want %q", got, want)
	}
}

func TestPrincipalOwnerIDPreservesClientOnlyLegacyOwnership(t *testing.T) {
	principal := Principal{ClientID: "legacy-service-client"}
	if got, want := principal.OwnerID(), "legacy-service-client"; got != want {
		t.Fatalf("OwnerID()=%q, want %q", got, want)
	}
}

func TestPrincipalOwnerIDFailsClosedForSubjectWithoutIssuer(t *testing.T) {
	principal := Principal{Subject: "user-123", ClientID: "crm-web-client"}
	if got := principal.OwnerID(); got != "" {
		t.Fatalf("OwnerID()=%q, want empty", got)
	}
}

func TestPrincipalOwnerIDEncodingAvoidsIssuerSubjectCollisions(t *testing.T) {
	first := Principal{Issuer: "issuer:a", Subject: "b:c"}
	second := Principal{Issuer: "issuer", Subject: "a:b"}
	if first.OwnerID() == second.OwnerID() {
		t.Fatal("different issuer/subject tuples must not collide")
	}
}

func TestPrincipalOwnerIDPreservesOpaqueWhitespaceBytes(t *testing.T) {
	principal := Principal{Issuer: " https://issuer.test ", Subject: " user-123 "}
	if got, want := principal.OwnerID(), "user:"+base64.RawURLEncoding.EncodeToString([]byte(" https://issuer.test "))+":"+base64.RawURLEncoding.EncodeToString([]byte(" user-123 ")); got != want {
		t.Fatalf("OwnerID()=%q, want canonical tuple %q", got, want)
	}
}
