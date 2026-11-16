package httpapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestJWKSVerifierAcceptsValidES256AndExtractsScopes(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	const kid = "test-key-01"
	jwks := ecJWK(privateKey, kid)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwks}})
	}))
	defer server.Close()
	verifier := &JWKSVerifier{issuer:"https://issuer.test", audience:"naeos-api", jwksURL:server.URL, client:server.Client()}
	now := time.Now().Unix()
	token := signedTestJWT(t, privateKey, kid, "ES256", map[string]any{
		"iss":verifier.issuer, "aud":verifier.audience, "sub":"user-123", "client_id":"svc-crm",
		"exp":now+300, "nbf":now-10, "naeos_scopes":[]string{"intelligence:read","intelligence:write"},
	})
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil { t.Fatal(err) }
	if principal.ClientID!="user-123" || !principal.Scopes["intelligence:read"] || !principal.Scopes["intelligence:write"] {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestJWKSVerifierIgnoresUntrustedScopeSources(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	verifier := &JWKSVerifier{issuer:"https://issuer.test", audience:"naeos-api", keys:map[string]*ecdsa.PublicKey{"kid":&privateKey.PublicKey}, fetchedAt:time.Now()}
	now := time.Now().Unix()
	claims := map[string]any{
		"iss":verifier.issuer, "aud":verifier.audience, "sub":"user-123", "exp":now+300,
		"scope":"intelligence:read intelligence:write",
		"scopes":[]string{"intelligence:read"},
		"user_metadata":map[string]any{"naeos_scopes":[]string{"intelligence:write"}},
	}
	token := signedTestJWT(t, privateKey, "kid", "ES256", claims)
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil { t.Fatal(err) }
	if len(principal.Scopes) != 0 { t.Fatalf("generic scope/scopes and user_metadata must not grant NAEOS capabilities: %+v", principal.Scopes) }
}

func TestJWKSVerifierAllowlistFiltersNAEOSScopes(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	verifier := &JWKSVerifier{issuer:"https://issuer.test", audience:"naeos-api", keys:map[string]*ecdsa.PublicKey{"kid":&privateKey.PublicKey}, fetchedAt:time.Now()}
	token := signedTestJWT(t, privateKey, "kid", "ES256", map[string]any{
		"iss":verifier.issuer, "aud":verifier.audience, "sub":"user-123", "exp":time.Now().Unix()+300,
		"naeos_scopes":[]string{"intelligence:read", "admin:*", "intelligence:write", "intelligence:read"},
	})
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil { t.Fatal(err) }
	if len(principal.Scopes) != 2 || !principal.Scopes["intelligence:read"] || !principal.Scopes["intelligence:write"] {
		t.Fatalf("unexpected allowlisted capability set: %+v", principal.Scopes)
	}
}

func TestJWKSVerifierRejectsWrongAlgorithmsAndExpiredClaims(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	verifier := &JWKSVerifier{issuer:"https://issuer.test", audience:"naeos-api", keys:map[string]*ecdsa.PublicKey{"kid":&privateKey.PublicKey}, fetchedAt:time.Now()}
	expired := signedTestJWT(t, privateKey, "kid", "ES256", map[string]any{"iss":verifier.issuer,"aud":verifier.audience,"sub":"svc","exp":time.Now().Unix()-1})
	if _,err:=verifier.Verify(context.Background(),expired); err==nil { t.Fatal("expired JWT must be rejected") }
	for _, alg := range []string{"RS256","HS256","none"} {
		token := unsignedOrSignedHeaderToken(t, privateKey, "kid", alg)
		if _,err:=verifier.Verify(context.Background(),token); err==nil { t.Fatalf("%s token must be rejected",alg) }
	}
	for _, rawHeader := range []string{`{"alg":"ES256","kid":"kid","crit":["unknown"]}`, `{"alg":"ES256","kid":"kid","b64":false}`} {
		token := base64.RawURLEncoding.EncodeToString([]byte(rawHeader)) + "." + base64.RawURLEncoding.EncodeToString([]byte(`{}`)) + "." + base64.RawURLEncoding.EncodeToString(make([]byte, 64))
		if _,err:=verifier.Verify(context.Background(),token); err==nil { t.Fatalf("unsupported critical JWT header must be rejected: %s",rawHeader) }
	}
}

func TestNewJWKSVerifierRequiresHTTPSAndCompleteConfig(t *testing.T) {
	if _,err:=NewJWKSVerifier("https://issuer","aud","http://jwks"); err==nil { t.Fatal("HTTP JWKS URL must be rejected") }
	if _,err:=NewJWKSVerifier("https://issuer","","https://jwks"); err==nil { t.Fatal("incomplete verifier configuration must be rejected") }
}

func TestJWKSVerifierAcceptsAudienceArrayAndRejectsFutureNotBefore(t *testing.T) {
	privateKey,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader); if err!=nil {t.Fatal(err)}
	verifier:=&JWKSVerifier{issuer:"https://issuer.test",audience:"naeos-api",keys:map[string]*ecdsa.PublicKey{"kid":&privateKey.PublicKey},fetchedAt:time.Now()}
	now:=time.Now().Unix()
	valid:=signedTestJWT(t,privateKey,"kid","ES256",map[string]any{"iss":verifier.issuer,"aud":[]string{"other-api",verifier.audience},"sub":"svc-test","exp":now+300,"nbf":now-1})
	if _,err:=verifier.Verify(context.Background(),valid); err!=nil {t.Fatalf("matching audience should be accepted: %v",err)}
	future:=signedTestJWT(t,privateKey,"kid","ES256",map[string]any{"iss":verifier.issuer,"aud":verifier.audience,"sub":"svc-test","exp":now+300,"nbf":now+60})
	if _,err:=verifier.Verify(context.Background(),future); err==nil {t.Fatal("future not-before must be rejected")}
}

func TestJWKSVerifierRejectsTamperedSignatureAndUnknownKeyID(t *testing.T) {
	privateKey,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader); if err!=nil {t.Fatal(err)}
	const kid="kid"
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_ = json.NewEncoder(w).Encode(map[string]any{"keys":[]any{ecJWK(privateKey,kid)}})}))
	defer server.Close()
	verifier:=&JWKSVerifier{issuer:"https://issuer.test",audience:"naeos-api",jwksURL:server.URL,client:server.Client()}
	token:=signedTestJWT(t,privateKey,kid,"ES256",map[string]any{"iss":verifier.issuer,"aud":verifier.audience,"sub":"svc-test","exp":time.Now().Unix()+300})
	parts:=strings.Split(token,".")
	parts[1]=base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"https://issuer.test","aud":"naeos-api","sub":"attacker","exp":9999999999}`))
	if _,err:=verifier.Verify(context.Background(),strings.Join(parts,".")); err==nil {t.Fatal("tampered claims must be rejected")}
	unknown:=signedTestJWT(t,privateKey,"unknown-kid","ES256",map[string]any{"iss":verifier.issuer,"aud":verifier.audience,"sub":"svc-test","exp":time.Now().Unix()+300})
	if _,err:=verifier.Verify(context.Background(),unknown); err==nil {t.Fatal("unknown kid must be rejected")}
}

func TestJWKSVerifierRejectsMissingOrMalformedRequiredClaims(t *testing.T) {
	privateKey,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader); if err!=nil {t.Fatal(err)}
	verifier:=&JWKSVerifier{issuer:"https://issuer.test",audience:"naeos-api",keys:map[string]*ecdsa.PublicKey{"kid":&privateKey.PublicKey},fetchedAt:time.Now()}
	base:=map[string]any{"iss":verifier.issuer,"aud":verifier.audience,"sub":"svc-test","exp":time.Now().Unix()+300}
	tests:=[]struct{name string; mutate func(map[string]any)}{
		{"missing expiry",func(c map[string]any){delete(c,"exp")}},
		{"missing issued-at",func(c map[string]any){c["iat"]=int64(0)}},
		{"future issued-at",func(c map[string]any){c["iat"]=time.Now().Unix()+120}},
		{"issued-at after expiry",func(c map[string]any){c["iat"]=time.Now().Unix()+600}},
		{"not-before after expiry",func(c map[string]any){c["nbf"]=time.Now().Unix()+600}},
		{"missing issuer",func(c map[string]any){delete(c,"iss")}},
		{"wrong issuer",func(c map[string]any){c["iss"]="https://attacker.test"}},
		{"wrong audience",func(c map[string]any){c["aud"]=[]string{"other-api"}}},
		{"missing identity",func(c map[string]any){delete(c,"sub")}},
		{"blank identity",func(c map[string]any){c["sub"]="  "}},
		{"malformed not-before",func(c map[string]any){c["nbf"]="not-a-timestamp"}},
	}
	for _,tt:=range tests {t.Run(tt.name,func(t *testing.T){
		claims:=make(map[string]any,len(base)); for k,v:=range base {claims[k]=v}; tt.mutate(claims)
		if _,err:=verifier.Verify(context.Background(),signedTestJWT(t,privateKey,"kid","ES256",claims)); err==nil {t.Fatal("invalid claims must be rejected")}
	})}
}

func TestJWKSVerifierRefreshesOnUnknownKeyIDForRotation(t *testing.T) {
	oldKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{ecJWK(newKey, "rotated-kid")}})
	}))
	defer server.Close()
	verifier := &JWKSVerifier{issuer:"https://issuer.test", audience:"naeos-api", jwksURL:server.URL, client:server.Client(),
		keys:map[string]*ecdsa.PublicKey{"old-kid":&oldKey.PublicKey}, fetchedAt:time.Now()}
	token := signedTestJWT(t, newKey, "rotated-kid", "ES256", map[string]any{
		"iss":verifier.issuer, "aud":verifier.audience, "sub":"user-123", "exp":time.Now().Unix()+300,
	})
	if _, err := verifier.Verify(context.Background(), token); err != nil {
		t.Fatalf("unknown kid should trigger refresh and accept the rotated trusted key: %v", err)
	}
	if requests != 1 { t.Fatalf("expected one JWKS refresh, got %d", requests) }
}

func TestJWKSVerifierFailsClosedWhenRefreshUnavailableEvenWithStaleCache(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	verifier := &JWKSVerifier{issuer:"https://issuer.test", audience:"naeos-api", jwksURL:server.URL, client:server.Client(),
		keys:map[string]*ecdsa.PublicKey{"kid":&privateKey.PublicKey}, fetchedAt:time.Now().Add(-jwksCacheTTL-time.Second)}
	token := signedTestJWT(t, privateKey, "kid", "ES256", map[string]any{
		"iss":verifier.issuer, "aud":verifier.audience, "sub":"user-123", "exp":time.Now().Unix()+300,
	})
	if _, err := verifier.Verify(context.Background(), token); err == nil {
		t.Fatal("expired JWKS cache must not be used when refresh fails")
	}
}

func ecJWK(key *ecdsa.PrivateKey,kid string) map[string]any {
	return map[string]any{"kty":"EC","use":"sig","alg":"ES256","kid":kid,"crv":"P-256",
		"x":base64.RawURLEncoding.EncodeToString(coordinateBytes(key.X)),
		"y":base64.RawURLEncoding.EncodeToString(coordinateBytes(key.Y)),
		"key_ops":[]string{"verify"}}
}

func signedTestJWT(t *testing.T,key *ecdsa.PrivateKey,kid,alg string,claims any) string {
	t.Helper()
	header,err:=json.Marshal(map[string]string{"alg":alg,"kid":kid,"typ":"JWT"}); if err!=nil {t.Fatal(err)}
	if claimMap,ok:=claims.(map[string]any); ok { if _,exists:=claimMap["iat"]; !exists { claimMap["iat"]=time.Now().Unix() } }
	payload,err:=json.Marshal(claims); if err!=nil {t.Fatal(err)}
	encodedHeader:=base64.RawURLEncoding.EncodeToString(header)
	encodedPayload:=base64.RawURLEncoding.EncodeToString(payload)
	input:=encodedHeader+"."+encodedPayload
	digest:=sha256.Sum256([]byte(input))
	r,s,err:=ecdsa.Sign(rand.Reader,key,digest[:]); if err!=nil {t.Fatal(err)}
	sig:=make([]byte,64); r.FillBytes(sig[:32]); s.FillBytes(sig[32:])
	return fmt.Sprintf("%s.%s",input,base64.RawURLEncoding.EncodeToString(sig))
}

func unsignedOrSignedHeaderToken(t *testing.T,key *ecdsa.PrivateKey,kid,alg string) string {
	t.Helper()
	header,_:=json.Marshal(map[string]string{"alg":alg,"kid":kid})
	payload,_:=json.Marshal(map[string]any{"iss":"https://issuer.test","aud":"naeos-api","sub":"svc","exp":time.Now().Unix()+300})
	input:=base64.RawURLEncoding.EncodeToString(header)+"."+base64.RawURLEncoding.EncodeToString(payload)
	if alg!="ES256" { return input+"."+base64.RawURLEncoding.EncodeToString(make([]byte,64)) }
	digest:=sha256.Sum256([]byte(input)); r,s,_:=ecdsa.Sign(rand.Reader,key,digest[:]); sig:=make([]byte,64); r.FillBytes(sig[:32]); s.FillBytes(sig[32:])
	return input+"."+base64.RawURLEncoding.EncodeToString(sig)
}

func coordinateBytes(value *big.Int) []byte {
	out := make([]byte, 32)
	value.FillBytes(out)
	return out
}
