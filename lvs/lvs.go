package lvs

import (
	"crypto/ecdsa"
	"crypto/sha512"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"hash"
	"io"
	"os"
)

var Fingerprint string
var PublicKey string

func VerifyLicenceKey(key []byte) (bool, error) {
	pubDecoded, err := decodePublicKey([]byte(PublicKey))
	if err != nil {
		return false, err
	}
	hash, err := CalculateExecutableHash()
	if err != nil {
		return false, err
	}

	return ecdsa.VerifyASN1(pubDecoded, hash, key), nil
}

func CalculateExecutableHash() ([]byte, error) {
	var h hash.Hash = sha512.New()

	executable, err := os.Executable()
	if err != nil {
		panic(err)
	}
	f, err := os.Open(executable)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func decodePublicKey(pemEncoded []byte) (*ecdsa.PublicKey, error) {
	blockPub, _ := pem.Decode(pemEncoded)
	x509Encoded := blockPub.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509Encoded)
	if err != nil {
		return nil, err
	}
	publicKey := genericPublicKey.(*ecdsa.PublicKey)

	return publicKey, nil
}
