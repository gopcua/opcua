//go:build conformance

// A USERNAME UserTokenPolicy with an empty securityPolicyUri means the
// SecureChannel's policy: on a Basic256Sha256 SignAndEncrypt channel, the
// client encrypts the password with Basic256Sha256 and names that policy's
// algorithm in the UserNameIdentityToken.
//
// The clause: OPC 10000-4 §7.41 UserTokenPolicy, securityPolicyUri: "The
// security policy to use when encrypting or signing the UserIdentityToken
// when it is passed to the Server in the ActivateSession request. Clause
// 7.40 describes how this parameter is used. The security policy for the
// SecureChannel is used if this value is null or empty." The conformance
// unit is OPC 10000-7's "Security User Name Password Client".
//
// Neighbouring rows: legacy_first_username_policies_use_the_endpoints_own_test.go
// scores an endpoint that lists several USERNAME policies, where this row's
// lists one. unverified_server_signature_fails_connect_test.go scores the
// handshake this row's ActivateSession request rides on. NOT-TESTED.md
// covers the clause's ECC and RSA_DH sentence.
//
// The row runs against the in-tree server's Basic256Sha256 SignAndEncrypt
// endpoint. It replaces the endpoint's UserIdentityTokens with a single
// USERNAME policy whose securityPolicyUri is empty, because the in-tree
// server never builds one (server/server.go:427-432). See harness.go.
//
// The row asserts that the recorded token carries the offered PolicyID and
// the rsa-oaep EncryptionAlgorithm, and that its Password decrypts, with
// crypto/rsa and the server's key, to [length][rowPass][serverNonce], the
// layout EncryptUserPassword builds (uasc/secure_channel_crypto.go:87-91).
// It asserts nothing about the UserName field, the nonce's own bytes, or
// whether Connect succeeded.
//
// gopcua passes this row. Both SecurityFromEndpoint (config.go:465-469) and
// EncryptUserPassword (uasc/secure_channel_crypto.go:67-70) fall back to the
// channel's policy, so the row stays green while either one does; the
// package README lists the mutants that turn it red. UA-.NETStandard,
// Eclipse Milo, open62541 and node-opcua all use the channel's policy here,
// read from source at their 2026-11-18 heads and not run.
package session

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"testing"

	"github.com/gopcua/opcua/ua"
)

// unnamedPolicyID is the PolicyID of the one USERNAME policy the row offers.
const unnamedPolicyID = "username_unnamed"

func TestEmptyTokenPolicyMeansTheChannelsPolicy(t *testing.T) {
	s, teardown, why := startSessionServer()
	t.Cleanup(teardown)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	ep, why, fromGopcua := s.endpoint(ua.SecurityPolicyURIBasic256Sha256, ua.MessageSecurityModeSignAndEncrypt)
	if why != "" {
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
		} else {
			t.Fatalf("could not set the test up: %s", why)
		}
		return
	}

	ep = withUsernameTokens(ep, &ua.UserTokenPolicy{
		PolicyID:  unnamedPolicyID,
		TokenType: ua.UserTokenTypeUserName,
	})

	waitConnect, why, fromGopcua := launchClient(s, ep, rowUser, rowPass)
	if why != "" {
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
		} else {
			t.Fatalf("could not set the test up: %s", why)
		}
		return
	}
	if !waitConnect() {
		t.Fatalf("could not set the test up: Connect did not return within %v", connectBound)
	}

	req, why := s.awaitActivateSession(afterConnectBound)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	tok, why, unscoreable := usernameToken(req)
	if why != "" {
		if unscoreable {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
		} else {
			t.Errorf("%s", why)
		}
		return
	}

	if tok.PolicyID != unnamedPolicyID {
		t.Errorf("the token carries PolicyID %q, not %q, the only USERNAME policy offered",
			tok.PolicyID, unnamedPolicyID)
	}
	if tok.EncryptionAlgorithm != rsaOAEPURI {
		t.Errorf("the token names EncryptionAlgorithm %q, not %q, the channel's Basic256Sha256",
			tok.EncryptionAlgorithm, rsaOAEPURI)
	}

	// A password in the clear, or encrypted under another policy, does not
	// decrypt here.
	plaintext, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, s.key, tok.Password, nil)
	if err != nil {
		t.Errorf("the Password does not decrypt as RSA-OAEP-SHA1 under the server's key: %v", err)
		return
	}
	secretLen := len(rowPass) + serverNonceLength
	if len(plaintext) != 4+secretLen {
		t.Errorf("the decrypted Password is %d bytes, not 4+%d+%d (length, password, nonce)",
			len(plaintext), len(rowPass), serverNonceLength)
		return
	}
	if prefix := binary.LittleEndian.Uint32(plaintext[:4]); prefix != uint32(secretLen) {
		t.Errorf("the decrypted Password's length prefix is %d, not %d", prefix, secretLen)
		return
	}
	if got := string(plaintext[4 : 4+len(rowPass)]); got != rowPass {
		t.Errorf("the decrypted Password is %q, not %q", got, rowPass)
		return
	}

	if t.Failed() {
		return
	}
	t.Logf("the client used policy %q and encrypted the password with the channel's %s",
		tok.PolicyID, tok.EncryptionAlgorithm)
}
