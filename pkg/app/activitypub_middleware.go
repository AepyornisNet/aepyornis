package app

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

const requestingActorContextKey = "requesting_actor"
const activityPubAcceptContentType = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`

type activityPubHTTPClient struct {
	client *http.Client
}

func (c activityPubHTTPClient) CtxGet(ctx context.Context, uri string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", activityPubAcceptContentType)

	return c.client.Do(req)
}

func (c activityPubHTTPClient) LoadActor(ctx context.Context, actorIRI string) (*vocab.Actor, error) {
	resp, err := c.CtxGet(ctx, actorIRI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("failed to load actor: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var actor vocab.Actor
	if err := jsonld.Unmarshal(body, &actor); err != nil {
		return nil, err
	}

	return &actor, nil
}

func verifyDigestHeader(req *http.Request) error {
	digestHeader := req.Header.Get("Digest")
	if digestHeader == "" {
		return nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	sha256Digest := ""
	for part := range strings.SplitSeq(digestHeader, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}

		algorithm := strings.TrimSpace(kv[0])
		value := strings.Trim(strings.TrimSpace(kv[1]), `"`)
		if strings.EqualFold(algorithm, "sha-256") {
			sha256Digest = value
			break
		}
	}

	if sha256Digest == "" {
		return errors.New("unsupported digest algorithm")
	}

	sum := sha256.Sum256(body)
	expected := base64.StdEncoding.EncodeToString(sum[:])
	if sha256Digest != expected {
		return errors.New("digest mismatch")
	}

	return nil
}

func verifyDateHeader(req *http.Request) error {
	rawDate := req.Header.Get("Date")
	if rawDate == "" {
		return errors.New("date header is required")
	}

	ts, err := time.Parse(time.RFC1123, rawDate)
	if err != nil {
		ts, err = time.Parse(time.RFC1123Z, rawDate)
		if err != nil {
			return errors.New("invalid date header")
		}
	}

	now := time.Now()
	if ts.Before(now.Add(-30*time.Second)) || ts.After(now.Add(30*time.Second)) {
		return errors.New("date header out of allowed window")
	}

	return nil
}

func parseSignatureHeader(sig string) map[string]string {
	result := map[string]string{}
	for part := range strings.SplitSeq(sig, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		result[kv[0]] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
	}

	return result
}

func signatureHeaderValue(req *http.Request) string {
	if sig := req.Header.Get("Signature"); sig != "" {
		return sig
	}

	authz := req.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authz), "signature ") {
		return strings.TrimSpace(authz[len("Signature "):])
	}

	return ""
}

func parseRSAPublicKey(p string) (*rsa.PublicKey, error) {
	blk, _ := pem.Decode([]byte(p))
	if blk == nil {
		return nil, errors.New("invalid public key pem")
	}

	if pubAny, err := x509.ParsePKIXPublicKey(blk.Bytes); err == nil {
		pub, ok := pubAny.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not rsa")
		}
		return pub, nil
	}

	pub, err := x509.ParsePKCS1PublicKey(blk.Bytes)
	if err != nil {
		return nil, err
	}

	return pub, nil
}

func actorIRIFromBody(body []byte) (string, error) {
	var activity vocab.Activity
	if err := jsonld.Unmarshal(body, &activity); err != nil {
		return "", err
	}

	if vocab.IsNil(activity.Actor) {
		return "", errors.New("actor not found")
	}

	if vocab.IsIRI(activity.Actor) {
		return activity.Actor.GetLink().String(), nil
	}

	var actorIRI string
	if err := vocab.OnActor(activity.Actor, func(actor *vocab.Actor) error {
		actorIRI = actor.ID.String()
		return nil
	}); err == nil && actorIRI != "" {
		return actorIRI, nil
	}

	return "", errors.New("invalid actor value")
}

func headerValueForSignature(req *http.Request, header string) string {
	header = strings.ToLower(header)
	if header == "(request-target)" {
		return strings.ToLower(req.Method) + " " + req.URL.RequestURI()
	}
	if header == "host" {
		return req.Host
	}
	return req.Header.Get(header)
}

type signatureParts struct {
	keyID        string
	headers      string
	signatureB64 string
}

func extractSignatureParts(req *http.Request) (*signatureParts, error) {
	sigRaw := signatureHeaderValue(req)
	if sigRaw == "" {
		return nil, nil
	}

	parsed := parseSignatureHeader(sigRaw)
	parts := &signatureParts{
		keyID:        parsed["keyId"],
		headers:      parsed["headers"],
		signatureB64: parsed["signature"],
	}

	if parts.keyID == "" || parts.headers == "" || parts.signatureB64 == "" {
		return nil, errors.New("invalid signature fields")
	}

	return parts, nil
}

func readRequestBody(req *http.Request) ([]byte, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func buildSigningPayload(req *http.Request, headersValue string) (string, error) {
	signedHeaders := strings.Fields(headersValue)
	required := map[string]bool{
		"(request-target)": false,
		"host":             false,
		"date":             false,
	}

	lines := make([]string, 0, len(signedHeaders))
	for _, h := range signedHeaders {
		lower := strings.ToLower(h)
		if _, ok := required[lower]; ok {
			required[lower] = true
		}

		v := headerValueForSignature(req, lower)
		if v == "" {
			return "", fmt.Errorf("signed header missing: %s", lower)
		}

		lines = append(lines, fmt.Sprintf("%s: %s", lower, v))
	}

	for h, present := range required {
		if !present {
			return "", fmt.Errorf("required signed header missing: %s", h)
		}
	}

	return strings.Join(lines, "\n"), nil
}

func decodeSignature(sigValue string) ([]byte, error) {
	sigBytes, err := base64.StdEncoding.DecodeString(sigValue)
	if err == nil {
		return sigBytes, nil
	}

	return base64.RawStdEncoding.DecodeString(sigValue)
}

func verifyHTTPSignature(req *http.Request, client activityPubHTTPClient) (*vocab.Actor, error) {
	parts, err := extractSignatureParts(req)
	if err != nil {
		return nil, err
	}
	if parts == nil {
		return nil, nil
	}

	if err := verifyDateHeader(req); err != nil {
		return nil, err
	}

	if err := verifyDigestHeader(req); err != nil {
		return nil, err
	}

	body, err := readRequestBody(req)
	if err != nil {
		return nil, err
	}

	actorIRI, err := actorIRIFromBody(body)
	if err != nil {
		return nil, err
	}

	if _, err := url.ParseRequestURI(actorIRI); err != nil {
		return nil, err
	}

	actor, err := client.LoadActor(req.Context(), actorIRI)
	if err != nil {
		return nil, err
	}

	if actor.PublicKey.ID.String() != parts.keyID {
		return nil, errors.New("signature key id mismatch")
	}

	toSign, err := buildSigningPayload(req, parts.headers)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256([]byte(toSign))

	sigBytes, err := decodeSignature(parts.signatureB64)
	if err != nil {
		return nil, err
	}

	pub, err := parseRSAPublicKey(actor.PublicKey.PublicKeyPem)
	if err != nil {
		return nil, err
	}

	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hash[:], sigBytes); err != nil {
		return nil, err
	}

	return actor, nil
}

func (a *App) RequestingActorMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		actor, err := verifyHTTPSignature(c.Request(), activityPubHTTPClient{client: http.DefaultClient})
		if err != nil {
			a.logger.Warn("invalid ActivityPub request signature", "error", err)
			return a.renderAPIV2Error(c, http.StatusBadRequest, dto.ErrBadRequest)
		}

		if actor != nil {
			c.Set(requestingActorContextKey, *actor)
		}

		return next(c)
	}
}
