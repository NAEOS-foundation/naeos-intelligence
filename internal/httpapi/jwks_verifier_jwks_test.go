package httpapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestJWKSVerifierRejectsDocumentsWithoutUsableSigningKeys(t *testing.T) {
	tests:=[]struct{name string; keys []map[string]string}{
		{"non EC key",map[string]string{"kty":"RSA","use":"sig","alg":"ES256","kid":"key-1","x":strings.Repeat("A",43),"y":strings.Repeat("A",43)}},
		{"wrong curve",map[string]string{"kty":"EC","use":"sig","alg":"ES256","kid":"key-1","crv":"P-384","x":strings.Repeat("A",43),"y":strings.Repeat("A",43)}},
		{"encryption key",map[string]string{"kty":"EC","use":"enc","alg":"ES256","kid":"key-1","crv":"P-256","x":strings.Repeat("A",43),"y":strings.Repeat("A",43)}},
		{"wrong algorithm",map[string]string{"kty":"EC","use":"sig","alg":"ES384","kid":"key-1","crv":"P-256","x":strings.Repeat("A",43),"y":strings.Repeat("A",43)}},
		{"missing key id",map[string]string{"kty":"EC","use":"sig","alg":"ES256","crv":"P-256","x":strings.Repeat("A",43),"y":strings.Repeat("A",43)}},
		{"invalid coordinates",map[string]string{"kty":"EC","use":"sig","alg":"ES256","kid":"key-1","crv":"P-256","x":"AQ","y":"AQ"}},
	}
	for _,tt:=range tests {t.Run(tt.name,func(t *testing.T){
		server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_ = json.NewEncoder(w).Encode(map[string]any{"keys":[]any{tt.keys}})})); defer server.Close()
		v:=&JWKSVerifier{issuer:"https://issuer.test",audience:"naeos-api",jwksURL:server.URL,client:server.Client()}
		if _,err:=v.key(context.Background(),"key-1"); err==nil {t.Fatal("JWKS without acceptable ES256 key must be rejected")}
	})}
}

func TestJWKSVerifierRejectsUnsuccessfulAndMalformedJWKSResponses(t *testing.T) {
	tests:=[]struct{name string; status int; body string}{
		{"HTTP error",http.StatusServiceUnavailable,""},
		{"malformed JSON",http.StatusOK,"{not-json"},
		{"missing keys member",http.StatusOK,"{}"},
	}
	for _,tt:=range tests {t.Run(tt.name,func(t *testing.T){
		server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(tt.status); _,_=w.Write([]byte(tt.body))})); defer server.Close()
		v:=&JWKSVerifier{issuer:"https://issuer.test",audience:"naeos-api",jwksURL:server.URL,client:server.Client()}
		if _,err:=v.key(context.Background(),"key-1"); err==nil {t.Fatal("bad JWKS response must be rejected")}
	})}
}

func TestJWKSVerifierRejectsOversizedJWKSResponse(t *testing.T) {
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_,_=w.Write([]byte(strings.Repeat(" ",maxJWKSBytes+1)))})); defer server.Close()
	v:=&JWKSVerifier{issuer:"https://issuer.test",audience:"naeos-api",jwksURL:server.URL,client:server.Client()}
	if _,err:=v.key(context.Background(),"key-1"); err==nil {t.Fatal("oversized JWKS must be rejected")}
}

func TestJWKSVerifierRefreshesWhenSigningKeyIDIsUnknown(t *testing.T) {
	oldKey,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader); if err!=nil {t.Fatal(err)}
	newKey,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader); if err!=nil {t.Fatal(err)}
	var requests atomic.Int32
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		key:=oldKey; kid:="key-1"; if requests.Add(1)>1 {key=newKey;kid="key-2"}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys":[]any{map[string]string{
			"kty":"EC","use":"sig","alg":"ES256","kid":kid,"crv":"P-256",
			"x":base64.RawURLEncoding.EncodeToString(key.X.BytesFixed(32)),
			"y":base64.RawURLEncoding.EncodeToString(key.Y.BytesFixed(32)),
		}}})
	})); defer server.Close()
	v:=&JWKSVerifier{issuer:"https://issuer.test",audience:"naeos-api",jwksURL:server.URL,client:server.Client()}
	if _,err:=v.key(context.Background(),"key-1"); err!=nil {t.Fatalf("initial key fetch failed: %v",err)}
	if _,err:=v.key(context.Background(),"key-2"); err!=nil {t.Fatalf("unknown kid should refresh JWKS: %v",err)}
	if got:=requests.Load(); got!=2 {t.Fatalf("expected 2 JWKS requests, got %d",got)}
}

func TestJWKSVerifierRejectsDuplicateKeyIDs(t *testing.T) {
	key1,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader); if err!=nil {t.Fatal(err)}
	key2,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader); if err!=nil {t.Fatal(err)}
	toJWK:=func(k *ecdsa.PrivateKey) map[string]string{return map[string]string{
		"kty":"EC","use":"sig","alg":"ES256","kid":"same","crv":"P-256",
		"x":base64.RawURLEncoding.EncodeToString(k.X.BytesFixed(32)),
		"y":base64.RawURLEncoding.EncodeToString(k.Y.BytesFixed(32)),
	}}
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_ = json.NewEncoder(w).Encode(map[string]any{"keys":[]any{toJWK(key1),toJWK(key2)}})})); defer server.Close()
	v:=&JWKSVerifier{issuer:"https://issuer.test",audience:"naeos-api",jwksURL:server.URL,client:server.Client()}
	if _,err:=v.key(context.Background(),"same"); err==nil {t.Fatal("duplicate key IDs must be rejected")}
}
