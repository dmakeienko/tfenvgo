package cmd

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	pgperrors "github.com/ProtonMail/go-crypto/openpgp/errors"
)

// Release signing keys, embedded so verification never depends on a key
// fetched from the same place as the release.
//
// hashicorp.asc is https://www.hashicorp.com/.well-known/pgp-key.txt, primary
// key fingerprint C874 011F 0AB4 0511 0D02 1055 3436 5D94 72D7 468F as listed
// on https://www.hashicorp.com/en/trust/security.
//
// opentofu.asc is https://get.opentofu.org/opentofu.asc, primary key
// fingerprint E3E6 E43D 84CB 852E ADB0 051D 0C0A F313 E5FD 9F80 as pinned by
// OpenTofu's official installer (https://get.opentofu.org/install-opentofu.sh).
//
// TestEmbeddedKeyFingerprints guards these files against accidental changes.
var (
	//go:embed keys/hashicorp.asc
	hashicorpPublicKey string
	//go:embed keys/opentofu.asc
	opentofuPublicKey string
)

const (
	hashicorpKeyFingerprint = "C874011F0AB405110D02105534365D9472D7468F"
	opentofuKeyFingerprint  = "E3E6E43D84CB852EADB0051D0C0AF313E5FD9F80"
)

// Armored public keys trusted to sign each flavor's SHA256SUMS. Variables so
// tests can substitute a throwaway key.
var (
	terraformSigningKey = hashicorpPublicKey
	tofuSigningKey      = opentofuPublicKey
)

// skipSignatureEnvKey disables signature verification when set to "1".
// Insecure: the SHA-256 check then only proves the archive matches a
// checksum file served from the same place.
const skipSignatureEnvKey = "TFENVGO_SKIP_SIGNATURE"

// maxSignatureFileSize caps the downloaded detached signature size.
const maxSignatureFileSize int64 = 64 * 1024

func signatureVerificationDisabled() bool {
	return os.Getenv(skipSignatureEnvKey) == "1"
}

// signingKey returns the armored public key trusted for flv's releases.
func signingKey(flv string) string {
	if flv == FlavorTofu {
		return tofuSigningKey
	}
	return terraformSigningKey
}

// verifySignature checks that sig is a valid detached OpenPGP signature of
// signed made by armoredKey. sig may be binary or ASCII-armored. It returns
// the signer's primary key fingerprint.
func verifySignature(armoredKey string, signed, sig []byte) (string, error) {
	keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armoredKey))
	if err != nil {
		return "", fmt.Errorf("failed to load signing key: %w", err)
	}

	check := openpgp.CheckDetachedSignature
	if bytes.HasPrefix(bytes.TrimSpace(sig), []byte("-----BEGIN PGP SIGNATURE-----")) {
		check = openpgp.CheckArmoredDetachedSignature
	}
	signer, err := check(keyring, bytes.NewReader(signed), bytes.NewReader(sig), nil)
	if err != nil {
		if errors.Is(err, pgperrors.ErrUnknownIssuer) {
			return "", errors.New("signature was not made by the trusted release key")
		}
		return "", fmt.Errorf("invalid signature: %w", err)
	}
	return fmt.Sprintf("%X", signer.PrimaryKey.Fingerprint), nil
}
