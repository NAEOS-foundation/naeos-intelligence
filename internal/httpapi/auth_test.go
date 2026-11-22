package httpapi

import (
	"encoding/base64"
	"testing"
)

func TestPrincipalOwnerIDUsesIssuerQualifiedSubject(t *testing.T) {
	p := Principal{Issuer: "https://issuer.example/auth/v1", Subject: "user-123", ClientID: "crm-web"}
	want := "user:" + base64.RawURLEncoding.EncodeToString([]byte(p.Issuer)) + ":" +
		base64.RawURLEncoding.EncodeToString([]byte(p.Subject))
	if got := p.OwnerID(); got != want {
		t.Fatalf("OwnerID()=%q, want %q", got, want)
	}
}

func TestPrincipalOwnerIDSeparatesIssuerNamespaces(t *testing.T) {
	first := Principal{Issuer: "https://issuer-a.example", Subject: "same-sub"}
	second := Principal{Issuer: "https://issuer-b.example", Subject: "same-sub"}
	if first.OwnerID() == second.OwnerID() {
		t.Fatal("identical subjects from different issuers must have different owner keys")
	}
}

func TestPrincipalOwnerIDRejectsMissingHumanIdentity(t *testing.T) {
	tests := []struct {
		name string
		p    Principal
	}{
		{name: "missing issuer", p: Principal{Subject: "user-123", ClientID: "service-client"}},
		{name: "missing subject", p: Principal{Issuer: "https://issuer.example", ClientID: "service-client"}},
		{name: "whitespace subject", p: Principal{Issuer: "https://issuer.example", Subject: "   ", ClientID: "service-client"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.OwnerID(); got != "" {
				t.Fatalf("OwnerID()=%q, want empty", got)
			}
		})
	}
}

func TestPrincipalOwnerIDRejectsOversizedCompositeKey(t *testing.T) {
	p := Principal{Issuer: "https://issuer.example/" + string(make([]byte, 150)), Subject: "user-123"}
	if got := p.OwnerID(); got != "" {
		t.Fatalf("oversized OwnerID()=%q, want empty", got)
	}
}
