package activitypub

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
)

const RequestingActorContextKey = "requesting_actor"
const ContentType = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`

func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return []byte{}, nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
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

func parseRSAPrivateKey(p string) (*rsa.PrivateKey, error) {
	blk, _ := pem.Decode([]byte(p))
	if blk == nil {
		return nil, errors.New("invalid private key pem")
	}

	if prvAny, err := x509.ParsePKCS8PrivateKey(blk.Bytes); err == nil {
		prv, ok := prvAny.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not rsa")
		}
		return prv, nil
	}

	prv, err := x509.ParsePKCS1PrivateKey(blk.Bytes)
	if err != nil {
		return nil, err
	}
	return prv, nil
}
