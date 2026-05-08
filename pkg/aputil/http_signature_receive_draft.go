package aputil

import (
	"errors"
	"fmt"
	"net/http"

	rfchttpsig "github.com/dadrus/httpsig"
	"github.com/go-fed/httpsig"
)

func VerifyDraftSignature(req *http.Request, resolver KeyResolver) (*RequestActor, error) {
	verifier, err := httpsig.NewVerifier(req)
	if err != nil {
		return nil, err
	}

	pubKey, err := resolver.ResolveKey(req.Context(), verifier.KeyId())
	if err != nil {
		return nil, fmt.Errorf("could not resolve key: %w", err)
	}

	alg, found := parseDraftAlgorithm(pubKey)
	if !found {
		return nil, errors.New("unsupported algorithm used in request signature")
	}

	if err := verifier.Verify(pubKey.Key, alg); err != nil {
		return nil, fmt.Errorf("could not verify key: %w", err)
	}

	actor := resolver.Actor()
	if actor == nil {
		return nil, errors.New("could not resolve actor from signature")
	}

	return &RequestActor{
		Actor: *actor,
	}, nil
}

func parseDraftAlgorithm(pubKey rfchttpsig.Key) (httpsig.Algorithm, bool) {
	switch pubKey.Algorithm {
	case rfchttpsig.RsaPkcs1v15Sha256, rfchttpsig.RsaPssSha256:
		return httpsig.RSA_SHA256, true
	case rfchttpsig.RsaPssSha384, rfchttpsig.RsaPkcs1v15Sha384:
		return httpsig.RSA_SHA384, true
	case rfchttpsig.RsaPkcs1v15Sha512, rfchttpsig.RsaPssSha512:
		return httpsig.RSA_SHA512, true
	case rfchttpsig.EcdsaP256Sha256:
		return httpsig.ECDSA_SHA256, true
	case rfchttpsig.EcdsaP384Sha384:
		return httpsig.ECDSA_SHA384, true
	case rfchttpsig.EcdsaP521Sha512:
		return httpsig.ECDSA_SHA512, true
	case rfchttpsig.Ed25519:
		return httpsig.ED25519, true
	case rfchttpsig.HmacSha256:
		return httpsig.HMAC_SHA256, true
	case rfchttpsig.HmacSha384:
		return httpsig.HMAC_SHA384, true
	case rfchttpsig.HmacSha512:
		return httpsig.HMAC_SHA512, true
	}

	return "", false
}
