package activitypub

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/activitypub/aptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// digestForBody
// ---------------------------------------------------------------------------

func TestDigestForBody_Format(t *testing.T) {
	body := []byte("hello world")
	sum := sha256.Sum256(body)
	expected := "SHA-256=" + base64.StdEncoding.EncodeToString(sum[:])

	assert.Equal(t, expected, digestForBody(body))
}

func TestDigestForBody_EmptyBody(t *testing.T) {
	d := digestForBody([]byte{})
	assert.True(t, strings.HasPrefix(d, "SHA-256="), "digest should start with SHA-256=")
}

// ---------------------------------------------------------------------------
// requestTarget
// ---------------------------------------------------------------------------

func TestRequestTarget(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com/inbox?foo=bar", nil)
	require.NoError(t, err)

	assert.Equal(t, "post /inbox?foo=bar", requestTarget(req))
}

// ---------------------------------------------------------------------------
// parseRSAPublicKey / parseRSAPrivateKey
// ---------------------------------------------------------------------------

func TestParseRSAKeyPair_RoundTrip(t *testing.T) {
	privateKeyPEM, publicKeyPEM := aptest.GenerateKeyPair(t)

	pub, err := parseRSAPublicKey(publicKeyPEM)
	require.NoError(t, err)
	assert.NotNil(t, pub)

	prv, err := parseRSAPrivateKey(privateKeyPEM)
	require.NoError(t, err)
	assert.NotNil(t, prv)

	// Keys must be consistent.
	assert.Equal(t, pub.N, prv.PublicKey.N)
}

func TestParseRSAPublicKey_InvalidPEM(t *testing.T) {
	_, err := parseRSAPublicKey("not a pem block")
	require.Error(t, err)
}

func TestParseRSAPrivateKey_InvalidPEM(t *testing.T) {
	_, err := parseRSAPrivateKey("not a pem block")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// readRequestBody
// ---------------------------------------------------------------------------

func TestReadRequestBody_ReadsAndRestores(t *testing.T) {
	original := []byte("request body content")
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/", bytes.NewReader(original))

	b1, err := readRequestBody(req)
	require.NoError(t, err)
	assert.Equal(t, original, b1)

	// Body must be readable a second time.
	b2, err := readRequestBody(req)
	require.NoError(t, err)
	assert.Equal(t, original, b2)
}

func TestReadRequestBody_NilBody(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	b, err := readRequestBody(req)
	require.NoError(t, err)
	assert.Empty(t, b)
}

// ---------------------------------------------------------------------------
// SignRequest – error cases
// ---------------------------------------------------------------------------

func TestSignRequest_NilRequest(t *testing.T) {
	err := SignRequest(nil, "key", "keyID")
	require.Error(t, err)
}

func TestSignRequest_MissingPrivateKey(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", nil)
	err := SignRequest(req, "", "keyID")
	require.Error(t, err)
}

func TestSignRequest_MissingKeyID(t *testing.T) {
	privateKeyPEM, _ := aptest.GenerateKeyPair(t)
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", nil)
	err := SignRequest(req, privateKeyPEM, "")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// SignRequest – happy path and resulting headers
// ---------------------------------------------------------------------------

func TestSignRequest_SetsExpectedHeaders(t *testing.T) {
	privateKeyPEM, _ := aptest.GenerateKeyPair(t)
	body := []byte(`{"type":"Follow"}`)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/inbox", bytes.NewReader(body))
	require.NoError(t, err)

	require.NoError(t, SignRequest(req, privateKeyPEM, "https://example.com/users/alice#main-key"))

	assert.NotEmpty(t, req.Header.Get("Date"))
	assert.NotEmpty(t, req.Header.Get("Digest"))
	assert.NotEmpty(t, req.Header.Get("Host"))
	assert.NotEmpty(t, req.Header.Get("Signature"))

	// Digest must match the body.
	expected := digestForBody(body)
	assert.Equal(t, expected, req.Header.Get("Digest"))

	// Signature header must include expected fields.
	sig := req.Header.Get("Signature")
	assert.Contains(t, sig, "keyId=")
	assert.Contains(t, sig, "headers=")
	assert.Contains(t, sig, "signature=")
	assert.Contains(t, sig, "algorithm=")
}

// ---------------------------------------------------------------------------
// parseSignatureHeader
// ---------------------------------------------------------------------------

func TestParseSignatureHeader(t *testing.T) {
	raw := `keyId="https://example.com/users/alice#main-key",headers="(request-target) host date digest",signature="abc123==",algorithm="rsa-sha256"`
	parsed := parseSignatureHeader(raw)

	assert.Equal(t, "https://example.com/users/alice#main-key", parsed["keyId"])
	assert.Equal(t, "(request-target) host date digest", parsed["headers"])
	assert.Equal(t, "abc123==", parsed["signature"])
	assert.Equal(t, "rsa-sha256", parsed["algorithm"])
}

// ---------------------------------------------------------------------------
// verifyDigestHeader
// ---------------------------------------------------------------------------

func TestVerifyDigestHeader_Valid(t *testing.T) {
	body := []byte("test body")
	sum := sha256.Sum256(body)
	digest := "SHA-256=" + base64.StdEncoding.EncodeToString(sum[:])

	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", bytes.NewReader(body))
	req.Header.Set("Digest", digest)

	require.NoError(t, verifyDigestHeader(req))
}

func TestVerifyDigestHeader_Mismatch(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", bytes.NewReader([]byte("real body")))
	req.Header.Set("Digest", "SHA-256=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")

	require.Error(t, verifyDigestHeader(req))
}

func TestVerifyDigestHeader_Missing(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", bytes.NewReader([]byte("body")))
	// No Digest header → no error (header is optional).
	require.NoError(t, verifyDigestHeader(req))
}

// ---------------------------------------------------------------------------
// verifyDateHeader
// ---------------------------------------------------------------------------

func TestVerifyDateHeader_Valid(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", nil)
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	require.NoError(t, verifyDateHeader(req))
}

func TestVerifyDateHeader_Missing(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", nil)
	require.Error(t, verifyDateHeader(req))
}

func TestVerifyDateHeader_TooOld(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", nil)
	old := time.Now().UTC().Add(-2 * time.Minute)
	req.Header.Set("Date", old.Format(time.RFC1123))
	require.Error(t, verifyDateHeader(req))
}

func TestVerifyDateHeader_InFuture(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", nil)
	future := time.Now().UTC().Add(2 * time.Minute)
	req.Header.Set("Date", future.Format(time.RFC1123))
	require.Error(t, verifyDateHeader(req))
}

// ---------------------------------------------------------------------------
// Sign + Verify integration round-trip
// ---------------------------------------------------------------------------

// TestSignAndVerifyRoundTrip signs a realistic inbox POST and verifies it using
// a mock HTTPS server that serves the actor document with the matching public key.
func TestSignAndVerifyRoundTrip(t *testing.T) {
	privateKeyPEM, publicKeyPEM := aptest.GenerateKeyPair(t)

	server, actorIRI, keyID := aptest.MockActorServer(t, "roundtrip", publicKeyPEM)
	aptest.UseTestTLSTransport(t, server)

	// Build a realistic POST payload that contains the actor IRI so that
	// VerifyRequest can extract it from the body.
	body := []byte(fmt.Sprintf(`{"@context":"https://www.w3.org/ns/activitystreams","type":"Follow","actor":"%s","object":"https://recipient.example.com/users/recipient"}`, actorIRI))

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"https://recipient.example.com/users/recipient/inbox",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	req.Host = "recipient.example.com"

	require.NoError(t, SignRequest(req, privateKeyPEM, keyID))

	requestingActor, err := VerifyRequest(req, server.Client())
	require.NoError(t, err)
	require.NotNil(t, requestingActor)
	assert.Equal(t, actorIRI, requestingActor.ID.String())
}

// TestVerifyRequest_NoSignatureHeader verifies that a request without any
// signature header returns (nil, nil) — i.e. unsigned requests are not rejected.
func TestVerifyRequest_NoSignatureHeader(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inbox", nil)
	actor, err := VerifyRequest(req, nil)
	require.NoError(t, err)
	assert.Nil(t, actor)
}
