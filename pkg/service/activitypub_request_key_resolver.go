package service

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/dadrus/httpsig"
	vocab "github.com/go-ap/activitypub"
	"gorm.io/gorm"
)

type activityPubRequestKeyResolver struct {
	db    *gorm.DB
	actor *vocab.Actor
}

func (r *activityPubRequestKeyResolver) ResolveKey(ctx context.Context, keyID string) (httpsig.Key, error) {
	actorIRI, err := actorIRIFromKeyID(keyID)
	if err != nil {
		return httpsig.Key{}, err
	}

	profile, err := r.findRemoteProfile(actorIRI)
	if err != nil {
		return httpsig.Key{}, err
	}

	if profile != nil && strings.TrimSpace(profile.PublicKey) != "" {
		r.actor = actorFromProfile(profile)
		return toHTTPsigKey(keyID, profile.PublicKey)
	}

	actor, err := aputil.LoadRemoteActor(ctx, actorIRI)
	if err != nil {
		return httpsig.Key{}, err
	}

	publicKeyPEM := strings.TrimSpace(actor.PublicKey.PublicKeyPem)
	if publicKeyPEM == "" {
		return httpsig.Key{}, errors.New("remote actor public key is empty")
	}

	remoteProfile, err := aputil.RemoteProfileFromActor(actor)
	if err != nil {
		return httpsig.Key{}, err
	}
	remoteProfile.PublicKey = publicKeyPEM

	if r.db != nil {
		if _, err := remoteProfile.UpsertRemote(r.db); err != nil {
			return httpsig.Key{}, err
		}
	}

	r.actor = actor

	return toHTTPsigKey(keyID, publicKeyPEM)
}

func (r *activityPubRequestKeyResolver) Actor() *vocab.Actor {
	return r.actor
}

func (r *activityPubRequestKeyResolver) findRemoteProfile(actorIRI string) (*model.Profile, error) {
	if r.db == nil {
		return nil, nil
	}

	profile := &model.Profile{}
	err := r.db.Where("url = ?", actorIRI).First(profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func actorIRIFromKeyID(keyID string) (string, error) {
	trimmed := strings.TrimSpace(keyID)
	if trimmed == "" {
		return "", errors.New("signature key id is empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}

	parsed.Fragment = ""
	actorIRI := strings.TrimSpace(parsed.String())
	if actorIRI == "" {
		return "", errors.New("invalid signature key id")
	}

	return actorIRI, nil
}

func actorFromProfile(profile *model.Profile) *vocab.Actor {
	if profile == nil {
		return nil
	}

	actorIRI := strings.TrimSpace(profile.ActorURL())
	if actorIRI == "" {
		return nil
	}

	actor := &vocab.Actor{ID: vocab.ID(actorIRI)}
	if profile.PublicKey != "" {
		actor.PublicKey = vocab.PublicKey{
			ID:           vocab.ID(actorIRI + "#main-key"),
			Owner:        vocab.IRI(actorIRI),
			PublicKeyPem: profile.PublicKey,
		}
	}

	return actor
}

func toHTTPsigKey(keyID, publicKeyPEM string) (httpsig.Key, error) {
	pubKey, alg, err := parsePublicKeyAndAlgorithm(publicKeyPEM)
	if err != nil {
		return httpsig.Key{}, err
	}

	return httpsig.Key{
		KeyID:     keyID,
		Algorithm: alg,
		Key:       pubKey,
	}, nil
}

func parsePublicKey(publicKeyPEM string) (crypto.PublicKey, error) {
	blk, _ := pem.Decode([]byte(publicKeyPEM))
	if blk == nil {
		return nil, errors.New("unable to decode PEM payload for public key")
	}

	if pubAny, err := x509.ParsePKIXPublicKey(blk.Bytes); err == nil && pubAny != nil {
		return pubAny, nil
	}

	if pub, err := x509.ParsePKCS1PublicKey(blk.Bytes); err == nil && pub != nil {
		return pub, nil
	}

	return nil, errors.New("invalid public key pem")
}

func parsePublicKeyAndAlgorithm(publicKeyPEM string) (crypto.PublicKey, httpsig.SignatureAlgorithm, error) {
	pkey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return nil, "", fmt.Errorf("could not parse public key: %w", err)
	}

	var key crypto.PublicKey
	var alg httpsig.SignatureAlgorithm
	switch pk := pkey.(type) {
	case *rsa.PublicKey:
		switch pk.Size() {
		case 128, 256:
			alg = httpsig.RsaPkcs1v15Sha256
		case 384:
			alg = httpsig.RsaPkcs1v15Sha384
		case 512:
			alg = httpsig.RsaPkcs1v15Sha512
		}
		key = pk
	case *ecdsa.PublicKey:
		if p := pk.Params(); p != nil {
			switch p.BitSize {
			case 128, 256:
				alg = httpsig.EcdsaP256Sha256
			case 384:
				alg = httpsig.EcdsaP384Sha384
			case 512:
				alg = httpsig.EcdsaP521Sha512
			}
		}
		key = pk
	case ed25519.PublicKey:
		alg = httpsig.Ed25519
		key = pk
	}
	return key, alg, nil
}
