//go:build conformance

// When an endpoint lists several USERNAME UserTokenPolicies with a legacy
// one first, the client uses the one a conforming server would have
// offered: the one naming the endpoint's own SecurityPolicy, or one naming
// none.
//
// The clauses: OPC 10000-4 §7.41 UserTokenPolicy. A server that lists
// several breaks "An EndpointDescription shall have no more than one
// USERNAME UserTokenPolicy and no more than one ISSUEDTOKEN UserTokenPolicy
// for each unique issuerEndpointUrl.", and real PLCs do (#912). The
// standard has no rule for choosing among them, so the row expects the
// policy this sentence would have left: "If the SecurityMode is not None,
// USERNAME and ISSUEDTOKEN UserTokenPolicies should specify the same
// SecurityPolicy as the EndpointDescription or should not explicitly
// specify a SecurityPolicy. If a SecurityPolicy is specified, it shall use
// the same PublicKey algorithm as the SecureChannel." The conformance unit
// is OPC 10000-7's "Security User Name Password Client".
//
// Neighbouring rows: empty_token_policy_means_the_channels_policy_test.go
// scores an endpoint with one unnamed policy and holds the decryption proof
// that the named algorithm is the one used, which this row relies on.
// unverified_server_signature_fails_connect_test.go scores the handshake
// this row's ActivateSession request rides on. NOT-TESTED.md covers the
// clause's ECC and RSA_DH sentence.
//
// The row runs against the in-tree server's Basic256Sha256 SignAndEncrypt
// endpoint in two arms. The first presents the server's own list unedited:
// username_basic128rsa15, then username_basic256sha256
// (server/server.go:400-435). The second replaces the list with a legacy
// policy followed by one naming none. See harness.go.
//
// Each arm asserts the recorded token's PolicyID and that its
// EncryptionAlgorithm is rsa-oaep, so a client that picks the right
// policy but encrypts it under the legacy one fails. The row asserts nothing about the anonymous policies or about the
// server's own conformance.
//
// gopcua passes this row since #914: SecurityFromEndpoint takes the first
// USERNAME policy that names the endpoint's own SecurityPolicy or none
// (config.go:436-447). Before it, both arms sent the legacy PolicyID with
// rsa-1_5.
//
// A known-good client fails this: Eclipse Milo's UsernameProvider takes the
// first usable USERNAME policy by default (ps -> ps.get(0)) and leaves the
// choice to the application. The row is kept because UA-.NETStandard
// prefers the policy matching the endpoint and warns about the duplicate,
// node-opcua drops Basic128Rsa15 when a server lists several (its comment
// names S7), and open62541's default build, which has no Basic128Rsa15,
// picks the matching one. A client that fakes the outcome by sending the
// right PolicyID under the legacy algorithm still fails both arms. All four
// stacks were read from source at their 2026-11-18 heads and not run.
package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gopcua/opcua/ua"
)

func TestLegacyFirstUsernamePoliciesUseTheEndpointsOwn(t *testing.T) {
	arms := []struct {
		name string
		// tokens replaces the endpoint's list; nil keeps the server's own.
		tokens       []*ua.UserTokenPolicy
		wantPolicyID string
	}{
		{
			name:         "a matching policy later in the list",
			wantPolicyID: "username_basic256sha256",
		},
		{
			name: "an empty-policy entry later in the list",
			tokens: []*ua.UserTokenPolicy{
				{
					PolicyID:          "username_legacy",
					TokenType:         ua.UserTokenTypeUserName,
					SecurityPolicyURI: ua.SecurityPolicyURIBasic128Rsa15,
				},
				{
					PolicyID:  "username_unnamed",
					TokenType: ua.UserTokenTypeUserName,
				},
			},
			wantPolicyID: "username_unnamed",
		},
	}

	for _, arm := range arms {
		t.Run(arm.name, func(t *testing.T) {
			o, teardown := runArm(arm.tokens)
			t.Cleanup(teardown)
			if o.failWhy != "" {
				t.Errorf("%s", o.failWhy)
				return
			}
			if o.setupWhy != "" {
				if o.fromGopcua {
					t.Errorf("gopcua's own behaviour made this run unscoreable: %s", o.setupWhy)
				} else {
					t.Fatalf("could not set the test up: %s", o.setupWhy)
				}
				return
			}

			got, want, conforming := scoreChosenPolicy(o, arm.wantPolicyID)
			if !conforming {
				t.Errorf("%s, not %s", got, want)
				return
			}
			t.Logf("%s - the conforming candidate", got)
		})
	}
}

// armOutcome is what one arm's run left to score. setupWhy is the reason
// the run cannot be scored, and fromGopcua reports whether gopcua caused it.
// failWhy is a plain failure: the client sent a token of the wrong type.
type armOutcome struct {
	token      *ua.UserNameIdentityToken
	policies   []*ua.UserTokenPolicy
	setupWhy   string
	fromGopcua bool
	failWhy    string
}

// runArm starts the server, presents its Basic256Sha256 SignAndEncrypt
// endpoint with tokens as its USERNAME list (the server's own list when
// tokens is nil), connects a client and records its token. The caller owns
// teardown, which is non-nil even when a stage failed.
func runArm(tokens []*ua.UserTokenPolicy) (o armOutcome, teardown func()) {
	s, teardown, why := startSessionServer()
	if why != "" {
		o.setupWhy = why
		return o, teardown
	}

	ep, why, fromGopcua := s.endpoint(ua.SecurityPolicyURIBasic256Sha256, ua.MessageSecurityModeSignAndEncrypt)
	if why != "" {
		o.setupWhy, o.fromGopcua = why, fromGopcua
		return o, teardown
	}
	if tokens != nil {
		ep = withUsernameTokens(ep, tokens...)
	}
	o.policies = usernamePolicies(ep)

	// The server's own list is gopcua's behaviour; a rewritten one is the
	// row's, so a wrong shape there declines.
	if why := stimulusShapeFailed(o.policies, ep); why != "" {
		o.setupWhy, o.fromGopcua = why, tokens == nil
		return o, teardown
	}

	waitConnect, why, fromGopcua := launchClient(s, ep, rowUser, rowPass)
	if why != "" {
		o.setupWhy, o.fromGopcua = why, fromGopcua
		return o, teardown
	}
	if !waitConnect() {
		o.setupWhy = fmt.Sprintf("Connect did not return within %v", connectBound)
		return o, teardown
	}

	req, why := s.awaitActivateSession(afterConnectBound)
	if why != "" {
		o.setupWhy = why
		return o, teardown
	}

	tok, why, unscoreable := usernameToken(req)
	if why != "" {
		if unscoreable {
			o.setupWhy, o.fromGopcua = why, true
		} else {
			o.failWhy = why
		}
		return o, teardown
	}
	o.token = tok
	return o, teardown
}

// usernamePolicies returns the endpoint's USERNAME policies in the order it
// lists them, which is the order SecurityFromEndpoint walks (config.go:436).
func usernamePolicies(ep *ua.EndpointDescription) []*ua.UserTokenPolicy {
	var out []*ua.UserTokenPolicy
	for _, t := range ep.UserIdentityTokens {
		if t.TokenType == ua.UserTokenTypeUserName {
			out = append(out, t)
		}
	}
	return out
}

// stimulusShapeFailed returns "" if pols holds the row's premise: at least
// two policies, the first naming a policy other than the endpoint's, and a
// later one naming the endpoint's or none. Otherwise it returns why not.
func stimulusShapeFailed(pols []*ua.UserTokenPolicy, ep *ua.EndpointDescription) string {
	if len(pols) < 2 {
		return fmt.Sprintf("the endpoint lists %d USERNAME policies, not the "+
			"several the row's premise needs", len(pols))
	}
	if pols[0].SecurityPolicyURI == "" || pols[0].SecurityPolicyURI == ep.SecurityPolicyURI {
		return fmt.Sprintf("the endpoint's first USERNAME policy names %q, "+
			"not a legacy policy the endpoint does not run", pols[0].SecurityPolicyURI)
	}
	for _, t := range pols[1:] {
		if t.SecurityPolicyURI == ep.SecurityPolicyURI || t.SecurityPolicyURI == "" {
			return ""
		}
	}
	return fmt.Sprintf("after the legacy policy %q, the endpoint lists no USERNAME "+
		"policy naming its own %q or none", pols[0].SecurityPolicyURI, ep.SecurityPolicyURI)
}

// scoreChosenPolicy describes the recorded token (got) and the expectation
// (want). conforming reports that the token carries wantPolicyID and
// rsa-oaep.
func scoreChosenPolicy(o armOutcome, wantPolicyID string) (got, want string, conforming bool) {
	got = fmt.Sprintf("the endpoint listed %s, and the client sent PolicyID %q and EncryptionAlgorithm %q",
		usernamePoliciesDesc(o.policies), o.token.PolicyID, o.token.EncryptionAlgorithm)
	want = fmt.Sprintf("PolicyID %q and EncryptionAlgorithm %q", wantPolicyID, rsaOAEPURI)
	conforming = o.token.PolicyID == wantPolicyID && o.token.EncryptionAlgorithm == rsaOAEPURI
	return got, want, conforming
}

// usernamePoliciesDesc formats pols as policyID="securityPolicyUri" pairs.
func usernamePoliciesDesc(pols []*ua.UserTokenPolicy) string {
	parts := make([]string, 0, len(pols))
	for _, t := range pols {
		uri := t.SecurityPolicyURI
		if uri == "" {
			uri = "(names none)"
		}
		parts = append(parts, fmt.Sprintf("%s=%q", t.PolicyID, uri))
	}
	return "USERNAME policies " + strings.Join(parts, ", ")
}
