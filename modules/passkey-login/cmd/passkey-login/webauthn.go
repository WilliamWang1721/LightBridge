package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

type clientData struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Origin    string `json:"origin"`
}

func parseClientDataJSON(b []byte) (clientData, error) {
	var cd clientData
	if err := json.Unmarshal(b, &cd); err != nil {
		return clientData{}, err
	}
	cd.Type = strings.TrimSpace(cd.Type)
	cd.Challenge = strings.TrimSpace(cd.Challenge)
	cd.Origin = strings.TrimSpace(cd.Origin)
	if cd.Type == "" || cd.Challenge == "" || cd.Origin == "" {
		return clientData{}, errors.New("clientDataJSON missing required fields")
	}
	return cd, nil
}

func verifyClientData(cd clientData, expectedType, expectedOrigin string, expectedChallenge []byte) error {
	if cd.Type != expectedType {
		return fmt.Errorf("unexpected clientData.type: %s", cd.Type)
	}
	if expectedOrigin != "" && cd.Origin != expectedOrigin {
		return fmt.Errorf("unexpected origin: %s", cd.Origin)
	}
	chal, err := base64urlDecode(cd.Challenge)
	if err != nil {
		return fmt.Errorf("invalid challenge encoding: %w", err)
	}
	if !bytes.Equal(chal, expectedChallenge) {
		return errors.New("challenge mismatch")
	}
	return nil
}

const (
	flagUserPresent    = 1 << 0
	flagUserVerified   = 1 << 2
	flagAttestedCred   = 1 << 6
	flagExtensionData  = 1 << 7
	minAuthDataLenBase = 32 + 1 + 4
)

type parsedAuthData struct {
	RPIDHash []byte
	Flags    byte
	SignCnt  uint32

	AAGUID      []byte
	Credential  []byte
	PublicKeyCB []byte // COSE_Key bytes
}

func parseAttestationObject(attObj []byte) ([]byte, error) {
	v, _, err := decodeCBORFrom(attObj)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[any]any)
	if !ok {
		return nil, errors.New("attestationObject must be a CBOR map")
	}
	raw, ok := m["authData"]
	if !ok {
		return nil, errors.New("attestationObject missing authData")
	}
	authData, ok := raw.([]byte)
	if !ok || len(authData) < minAuthDataLenBase {
		return nil, errors.New("invalid authData")
	}
	return authData, nil
}

func parseAuthenticatorData(authData []byte) (parsedAuthData, error) {
	if len(authData) < minAuthDataLenBase {
		return parsedAuthData{}, errors.New("authenticatorData too short")
	}
	out := parsedAuthData{
		RPIDHash: append([]byte(nil), authData[:32]...),
		Flags:    authData[32],
		SignCnt:  binary.BigEndian.Uint32(authData[33:37]),
	}
	return out, nil
}

func parseAttestedAuthData(authData []byte) (parsedAuthData, error) {
	base, err := parseAuthenticatorData(authData)
	if err != nil {
		return parsedAuthData{}, err
	}
	if base.Flags&flagAttestedCred == 0 {
		return parsedAuthData{}, errors.New("missing attested credential data")
	}
	rest := authData[minAuthDataLenBase:]
	if len(rest) < 16+2 {
		return parsedAuthData{}, errors.New("attested credential data too short")
	}
	base.AAGUID = append([]byte(nil), rest[:16]...)
	credIDLen := int(binary.BigEndian.Uint16(rest[16:18]))
	rest = rest[18:]
	if credIDLen <= 0 || credIDLen > len(rest) {
		return parsedAuthData{}, errors.New("invalid credential id length")
	}
	base.Credential = append([]byte(nil), rest[:credIDLen]...)
	rest = rest[credIDLen:]
	if len(rest) == 0 {
		return parsedAuthData{}, errors.New("missing credential public key")
	}
	_, consumed, err := decodeCBORFrom(rest)
	if err != nil {
		return parsedAuthData{}, fmt.Errorf("decode credentialPublicKey: %w", err)
	}
	if consumed <= 0 || consumed > len(rest) {
		return parsedAuthData{}, errors.New("invalid credentialPublicKey length")
	}
	base.PublicKeyCB = append([]byte(nil), rest[:consumed]...)
	return base, nil
}

func verifyRPIDHash(got []byte, rpID string) error {
	rpID = strings.TrimSpace(rpID)
	if rpID == "" {
		return errors.New("missing rpId")
	}
	sum := sha256.Sum256([]byte(rpID))
	if len(got) != len(sum) || !bytes.Equal(got, sum[:]) {
		return errors.New("rpIdHash mismatch")
	}
	return nil
}

func parseCOSEES256PublicKey(coseKey []byte) (*ecdsa.PublicKey, error) {
	v, _, err := decodeCBORFrom(coseKey)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[any]any)
	if !ok {
		return nil, errors.New("cose key must be map")
	}
	// COSE_Key parameters.
	// https://www.iana.org/assignments/cose/cose.xhtml
	getInt := func(k int64) (int64, bool) {
		if vv, ok := m[k]; ok {
			if i, ok := vv.(int64); ok {
				return i, true
			}
		}
		return 0, false
	}
	getBytes := func(k int64) ([]byte, bool) {
		if vv, ok := m[k]; ok {
			if b, ok := vv.([]byte); ok {
				return b, true
			}
		}
		return nil, false
	}

	kty, _ := getInt(1)
	alg, _ := getInt(3)
	crv, _ := getInt(-1)
	if kty != 2 { // EC2
		return nil, fmt.Errorf("unsupported kty: %d", kty)
	}
	if alg != -7 { // ES256
		return nil, fmt.Errorf("unsupported alg: %d", alg)
	}
	if crv != 1 { // P-256
		return nil, fmt.Errorf("unsupported crv: %d", crv)
	}
	xBytes, okX := getBytes(-2)
	yBytes, okY := getBytes(-3)
	if !okX || !okY || len(xBytes) == 0 || len(yBytes) == 0 {
		return nil, errors.New("missing x/y")
	}
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)
	if !elliptic.P256().IsOnCurve(x, y) {
		return nil, errors.New("public key not on P-256 curve")
	}
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, nil
}

func verifyAssertionSignature(pub *ecdsa.PublicKey, authenticatorData, clientDataJSON, signature []byte) error {
	if pub == nil {
		return errors.New("missing public key")
	}
	if len(authenticatorData) < minAuthDataLenBase {
		return errors.New("authenticatorData too short")
	}
	cdHash := sha256.Sum256(clientDataJSON)
	signed := make([]byte, 0, len(authenticatorData)+len(cdHash))
	signed = append(signed, authenticatorData...)
	signed = append(signed, cdHash[:]...)
	h := sha256.Sum256(signed)
	if !ecdsa.VerifyASN1(pub, h[:], signature) {
		return errors.New("invalid signature")
	}
	return nil
}

func isUserPresent(flags byte) bool { return flags&flagUserPresent != 0 }

func isUserVerified(flags byte) bool { return flags&flagUserVerified != 0 }

func hasExtensions(flags byte) bool { return flags&flagExtensionData != 0 }

