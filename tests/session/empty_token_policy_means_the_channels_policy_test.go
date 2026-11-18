//go:build conformance

// A USERNAME UserTokenPolicy whose securityPolicyUri is empty means the
// SecureChannel's own policy: on a channel secured with Basic256Sha256 in
// mode SignAndEncrypt, the client encrypts the password with Basic256Sha256,
// and the UserNameIdentityToken it sends names the channel policy's
// algorithm.
//
// The clauses: OPC 10000-4 §7.41 UserTokenPolicy, securityPolicyUri, the
// whole field: "The security policy to use when encrypting or signing the
// UserIdentityToken when it is passed to the Server in the ActivateSession
// request. Clause 7.40 describes how this parameter is used. The security
// policy for the SecureChannel is used if this value is null or empty."
// The conformance unit is OPC 10000-7's "Security User Name Password
// Client" (User Token - User Name Password Client Facet). The algorithm
// URI the row expects is Part 7's profile table for Basic256Sha256, which
// the tree quotes at uapolicy/policyBasic256Sha256.go:38-42:
// "AsymmetricEncryptionAlgorithm_RSA-OAEP-SHA1 		The RSA encryption
// algorithm which is defined in https://tools.ietf.org/html/rfc3447. The
// RSAES-OAEP scheme is used. The hash algorithm is SHA1 [...] The URI is
// http://www.w3.org/2001/04/xmlenc#rsa-oaep." The literal is written out
// in the row and never read from uapolicy, whose EncryptionURI() would let
// the library grade itself.
//
// Neighbouring rows:
// legacy_first_username_policies_use_the_endpoints_own_test.go scores the
// same field when an endpoint offers several USERNAME policies at once,
// where this row's endpoint offers exactly one; the None-mode sentence of
// the same clause - the ECC and RSA_DH prohibition - has no row, and
// NOT-TESTED.md names it and the reason.
//
// The row runs against the in-tree gopcua server from harness.go, on its
// Basic256Sha256 SignAndEncrypt endpoint, with a real gopcua client
// connected over the wire: the row fetches the server's own endpoint
// description with a real GetEndpoints, rewrites nothing but its
// UserIdentityTokens - to the one USERNAME policy with an empty
// securityPolicyUri, the narrowest surgery that presents the list, because
// the in-tree server's initEndpoints never emits a token policy with an
// empty SecurityPolicyURI (server/server.go:427-432) - and hands the rewritten
// description to the client the harness launches with
// SecurityFromEndpoint and AuthUsername. The server's ActivateSession
// handler is the harness's wrapper, which records the request and answers
// it Good; the in-tree handler checks no user credentials at all
// (server/session_service.go), so the recorded request is the only place
// the behaviour shows. See harness.go.
//
// The row asserts, from the recorded ActivateSession request, that the
// UserIdentityToken is the UserNameIdentityToken the client was configured
// to send, that its PolicyID is the single policy the endpoint offered -
// so the client used the policy it was handed - that its
// EncryptionAlgorithm is http://www.w3.org/2001/04/xmlenc#rsa-oaep,
// Basic256Sha256's user-token encryption URI, and that the Password
// really is that algorithm's ciphertext: the row decrypts it with the
// standard library's crypto/rsa.DecryptOAEP under SHA-1 and the server's
// minted key - never with gopcua code - and the plaintext must carry the
// row's own password, under the [four-byte length][password][serverNonce]
// layout the client's EncryptUserPassword builds
// (uasc/secure_channel_crypto.go:87-91) and whose length prefix therefore
// counts the row's password plus the 32-byte nonce the in-tree server
// minted (server/session_service.go:18). Whether Connect succeeded is a
// symptom the row does not score; a password that does not decrypt, or
// decrypts to anything else, is an ordinary failure, and so is an
// EncryptionAlgorithm that names another policy. It declines, and does
// not score, when no ActivateSession request arrives within the harness's
// afterConnectBound grace - a client that never sent one and a server
// that never answered its half land in the same reason - and on the
// harness's own setup failures, with the gopcua-side ones scored as
// gopcua's behaviour.
// It deliberately asserts nothing about the token's UserName field, about
// the server nonce's own bytes - that is the server's half of §7.40, not
// the client's choice of policy - or about the anonymous policies the
// rewrite removed; and it scores nothing about Connect's own error.
//
// gopcua passes this row as it stands: measured on this tree, the
// recorded request carried the offered PolicyID, the rsa-oaep
// EncryptionAlgorithm and a password that decrypted to the row's own.
// Two library paths produce that observable together, and the row scores
// what reaches the server, so it stays green while either one runs:
// SecurityFromEndpoint reads the empty securityPolicyUri and sets the
// session's auth policy to the endpoint's own (config.go:448-452), and
// EncryptUserPassword falls back to the channel's own policy when the
// auth policy it is handed is empty (uasc/secure_channel_crypto.go:67-70).
// The package README's ledger holds the measured one-line mutants and
// which of them turn the row red.
// UA-.NETStandard, Eclipse Milo, open62541 and node-opcua all encrypt a
// token whose securityPolicyUri is empty under the SecureChannel's policy,
// read from source at their 2026-11-18 heads and not run here.
package session

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"testing"

	"github.com/gopcua/opcua/ua"
)

const (
	// unnamedPolicyID is the PolicyID of the one USERNAME policy the row's
	// endpoint offers, the empty-securityPolicyUri one the clause is
	// about. The row chose the string, so the recorded request's PolicyID
	// can only name a policy the row itself wrote out.
	unnamedPolicyID = "username_unnamed"
)

func TestEmptyTokenPolicyMeansTheChannelsPolicy(t *testing.T) {
	s, teardown, why := startSessionServer()
	t.Cleanup(teardown)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	// endpoint's every failure is gopcua's own - both halves of the
	// exchange it runs are the library under test, per its comment in
	// harness.go - so the row scores it as gopcua's behaviour and never
	// as a decline.
	ep, why := s.endpoint(ua.SecurityPolicyURIBasic256Sha256, ua.MessageSecurityModeSignAndEncrypt)
	if why != "" {
		t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
		return
	}

	// The stimulus: the real endpoint, carrying exactly one USERNAME
	// policy, and its securityPolicyUri empty - the "null or empty" of the
	// clause. Everything else about the description stays the server's
	// own, per the header.
	ep = withUsernameTokens(ep, &ua.UserTokenPolicy{
		PolicyID:  unnamedPolicyID,
		TokenType: ua.UserTokenTypeUserName,
		// SecurityPolicyURI stays the empty string.
	})

	waitConnect, why, fromGopcua := launchClient(s, ep, rowUser, rowPass)
	if why != "" {
		// The certificate the harness mints is the fixture's half and
		// declines; a client opcua.NewClient would not build is the
		// library under test and scores as gopcua's behaviour, per
		// launchClient's comment in harness.go.
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
		} else {
			t.Fatalf("could not set the test up: %s", why)
		}
		return
	}
	// Connect has returned - or outlived its bound - so every request the
	// client was going to send is in; the teardown that follows cannot
	// close the server underneath it, per harness.go.
	waitConnect()

	// The wrapper records the request before it answers it, and Connect
	// does not return before the answer arrives, so the request is already
	// in the channel; afterConnectBound is a grace for the handoff and not
	// a second connectBound's wait for a request that can no longer arrive.
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

	// The PolicyID names the policy the client used. Only one was on
	// offer, so a PolicyID that is not this one means the client did not
	// use the policy it was handed at all.
	if tok.PolicyID != unnamedPolicyID {
		t.Errorf("the UserNameIdentityToken carries PolicyID %q, not %q, "+
			"the only USERNAME policy the endpoint offered",
			tok.PolicyID, unnamedPolicyID)
	}

	// The EncryptionAlgorithm names the channel policy's own - the
	// clause's "The security policy for the SecureChannel is used if this
	// value is null or empty."
	if tok.EncryptionAlgorithm != rsaOAEPURI {
		t.Errorf("the UserNameIdentityToken names EncryptionAlgorithm %q, "+
			"not %q, the user-token encryption URI of the channel's own "+
			"Basic256Sha256, which the empty securityPolicyUri of the policy "+
			"it used hands the client", tok.EncryptionAlgorithm, rsaOAEPURI)
	}

	// And the Password really is that algorithm's ciphertext: decrypted
	// with the standard library under the server's minted key, it carries
	// the row's own password under the layout the header names. A
	// password in the clear, or encrypted under any other policy, does
	// not decrypt here.
	plaintext, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, s.key, tok.Password, nil)
	if err != nil {
		t.Errorf("the Password does not decrypt as one RSA-OAEP-SHA1 block "+
			"under the server's key: %v; the clause hands the client the "+
			"channel's own Basic256Sha256 for this token, and the ciphertext "+
			"is not that policy's", err)
		return
	}
	wantLen := uint32(len(rowPass) + serverNonceLength)
	if len(plaintext) != 4+len(rowPass)+serverNonceLength {
		t.Errorf("the decrypted Password is %d bytes, not the %d of the "+
			"[four-byte length][password][serverNonce] layout that carries %d bytes "+
			"of password and the %d-byte nonce the server minted",
			len(plaintext), 4+len(rowPass)+serverNonceLength, len(rowPass), serverNonceLength)
		return
	}
	if prefix := binary.LittleEndian.Uint32(plaintext[:4]); prefix != wantLen {
		t.Errorf("the decrypted Password's length prefix is %d, not %d - the "+
			"count of the row's %d password bytes and the %d-byte nonce the "+
			"server minted - so the plaintext is not the password the client "+
			"encrypted",
			prefix, wantLen, len(rowPass), serverNonceLength)
		return
	}
	if string(plaintext[4:4+len(rowPass)]) != rowPass {
		t.Errorf("the decrypted Password does not carry the password the " +
			"row configured the client with, so the ciphertext encrypts " +
			"something else")
		return
	}

	if t.Failed() {
		return
	}
	t.Logf("the client used policy %q and encrypted the password with the "+
		"channel's own Basic256Sha256 (%s), and the plaintext carried the "+
		"row's password under the length prefix %d",
		tok.PolicyID, tok.EncryptionAlgorithm, wantLen)
}
