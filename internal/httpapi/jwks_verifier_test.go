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
	token := signedTestJWT(t, privateKey, kid, map[string]any{
		"iss":verifier.issuer, "aud":verifier.audience, "client_id":"svc-crm",
		"exp":now+300, "nbf":now-10, "scope":"intelligence:read intelligence:write",
	})
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil { t.Fatal(err) }
	if principal.ClientID!="svc-crm" || !principal.Scopes["intelligence:read"] || !principal.Scopes["intelligence:write"] {
		t.Fatalf("unexpected principal: %+v", principal)
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

func ecJWK(key *ecdsa.PrivateKey,kid string) map[string]any {
	return map[string]any{"kty":"EC","use":"sig","alg":"ES256","kid":kid,"crv":"P-256",
		"x":base64.RawURLEncoding.EncodeToString(key.X.BytesFixed(32)),
		"y":base64.RawURLEncoding.EncodeToString(key.Y.BytesFixed(32)),
		"key_ops":[]string{"verify"}}
}

func signedTestJWT(t *testing.T,key *ecdsa.PrivateKey,kid,alg string,claims any) string {
	t.Helper()
	header,err:=json.Marshal(map[string]string{"alg":alg,"kid":kid,"typ":"JWT"}); if err!=nil {t.Fatal(err)}
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
