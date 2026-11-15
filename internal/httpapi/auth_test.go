package httpapi

import "testing"

func TestPrincipalOwnerIDSeparatesHumanSubjectFromClientIdentity(t *testing.T) {
	principal := Principal{
		Issuer:   "https://issuer.example/auth/v1",
		Subject:  "user-123",
		ClientID: "crm-web-client",
	}
	if got, want := principal.OwnerID(), "user:https://issuer.example/auth/v1:user-123"; got != want {
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
