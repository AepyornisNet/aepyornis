package activitypub

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func digestForBody(body []byte) string {
	sum := sha256.Sum256(body)
	return "SHA-256=" + base64.StdEncoding.EncodeToString(sum[:])
}

func requestTarget(req *http.Request) string {
	return strings.ToLower(req.Method) + " " + req.URL.RequestURI()
}

func dateHeaderValue() string {
	return time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
}

func SignRequest(req *http.Request, privateKeyPEM, keyID string) error {
	if req == nil {
		return errors.New("nil request")
	}
	if privateKeyPEM == "" {
		return errors.New("missing private key")
	}
	if keyID == "" {
		return errors.New("missing key id")
	}

	body, err := readRequestBody(req)
	if err != nil {
		return err
	}

	digest := digestForBody(body)
	date := dateHeaderValue()
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}

	req.Header.Set("Date", date)
	req.Header.Set("Digest", digest)
	req.Header.Set("Host", host)

	toSign := strings.Join([]string{
		"(request-target): " + requestTarget(req),
		"host: " + host,
		"date: " + date,
		"digest: " + digest,
	}, "\n")

	h := sha256.Sum256([]byte(toSign))
	prv, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return err
	}

	sig, err := rsa.SignPKCS1v15(rand.Reader, prv, crypto.SHA256, h[:])
	if err != nil {
		return err
	}

	req.Header.Set("Signature", fmt.Sprintf(
		"keyId=\"%s\",headers=\"(request-target) host date digest\",signature=\"%s\",algorithm=\"rsa-sha256\"",
		keyID,
		base64.StdEncoding.EncodeToString(sig),
	))

	return nil
}
