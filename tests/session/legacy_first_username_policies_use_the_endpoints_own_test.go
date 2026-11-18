//go:build conformance

// A client meeting an endpoint that lists several USERNAME
// UserTokenPolicies - a list the standard forbids a server to build, and
// one real PLCs build anyway, naming a legacy policy first - uses the one
// a conforming server would have offered: the one that specifies the same
// SecurityPolicy as the endpoint, or the one that specifies none.
//
// The clauses: OPC 10000-4 §7.41 UserTokenPolicy. The server breaks "An
// EndpointDescription shall have no more than one USERNAME UserTokenPolicy
// and no more than one ISSUEDTOKEN UserTokenPolicy for each unique
// issuerEndpointUrl.", and the standard has no rule for choosing among
// the policies such a server lists, so no clause binds the client there;
// what it does say is which one a conforming server would have offered,
// and that sentence is where the row's expectation comes from: "If the
// SecurityMode is not None, USERNAME and ISSUEDTOKEN UserTokenPolicies
// should specify the same SecurityPolicy as the EndpointDescription or
// should not explicitly specify a SecurityPolicy. If a SecurityPolicy is
// specified, it shall use the same PublicKey algorithm as the
// SecureChannel." The conformance unit is OPC 10000-7's "Security User
// Name Password Client" (User Token - User Name Password Client Facet).
// A client that instead takes the first policy in the list works against
// the gopcua server and fails against the PLC that lists its legacy
// policy first, and the row exists to hold that line. The expected
// algorithm URI is written out in the row and never read from uapolicy,
// whose EncryptionURI() would let the library grade itself.
//
// Neighbouring rows:
// empty_token_policy_means_the_channels_policy_test.go scores the
// endpoint that offers the one unnamed policy, where this row's offers
// several - and holds the byte-level proof that the named algorithm is
// the one the ciphertext really uses, which this row's arms rest on and
// do not repeat; the None-mode sentence of the same clause - the ECC and
// RSA_DH prohibition - has no row, and NOT-TESTED.md names it and the
// reason.
//
// The row runs against the in-tree gopcua server from harness.go, in two
// arms. The first presents the server's own endpoint description,
// unedited: the harness enables Basic128Rsa15 before Basic256Sha256, so
// the Basic256Sha256 endpoint lists the USERNAME policy
// username_basic128rsa15 first and its own username_basic256sha256 after
// it (server/server.go:400-435) - the in-tree server breaks the
// no-more-than-one sentence by construction, and that list is the row's
// stimulus. The second rewrites the UserIdentityTokens of the same real
// description - everything else about it stays the server's own, the
// narrowest surgery, per harness.go - to a legacy policy first and one
// with an empty securityPolicyUri after it, the other shape the
// should-sentence names. Both arms connect a real gopcua client over the
// wire, launched by the harness with SecurityFromEndpoint and
// AuthUsername; the server's ActivateSession handler is the harness's
// wrapper, which records the request and answers it Good. See harness.go.
//
// The row asserts, from each arm's recorded ActivateSession request, the
// UserIdentityToken's PolicyID and the EncryptionAlgorithm it names: the
// arm's conforming candidate, and the channel policy's rsa-oaep URI. The
// first arm expects username_basic256sha256, the matching policy the
// server listed after the legacy one; the second expects
// username_unnamed, the policy that names none, which the should-sentence
// hands to the channel's own policy. Whether Connect succeeded is a
// symptom the row does not score. It declines when no ActivateSession
// request arrives within the harness's afterConnectBound grace or the
// harness's own setup fails; a gopcua-side setup failure scores as
// gopcua's behaviour, and the same split scores the server's own list
// when it does not hold the several USERNAME policies the premise
// needs, while a rewritten list that does not is the arm's own
// construction and declines; a recorded token of the wrong type is a
// failure the arm scores outright, not a decline. The knownBug arm
// keys on exactly the defect the constant names: a request that
// arrived carrying the first, legacy policy's PolicyID and that
// policy's rsa-1_5 algorithm - a presence, so a client that never
// reaches the stimulus declines and does not land in the green arm -
// and any other wrong outcome, the right PolicyID with the wrong
// algorithm above all, is a hard failure even while the constant
// stands; the deletion arm fires on a request that named the right
// one. It deliberately asserts nothing
// about the anonymous policies the endpoint also lists, nothing about
// the server's own conformance - the no-more-than-one sentence binds the
// server, and the row scores the client a non-conforming server
// confronts.
//
// gopcua fails this row as it stands: measured on this tree,
// SecurityFromEndpoint walks the endpoint's UserIdentityTokens in order
// and returns on the first whose TokenType matches (config.go:429-453),
// so both arms sent the legacy policy's PolicyID and encrypted the
// password with http://www.w3.org/2001/04/xmlenc#rsa-1_5, Basic128Rsa15's
// user-token URI. The row carries a knownBug constant for that; the
// clause's should-sentence is the expectation, and the first-match walk
// is the defect.
//
// A known-good client fails this: Eclipse Milo's UsernameProvider takes the
// first usable USERNAME policy by default (ps -> ps.get(0)) and leaves the
// choice to the application. The test is kept because UA-.NETStandard
// returns the policy whose SecurityPolicyUri equals the endpoint's ahead of
// list position and warns about the duplicate, node-opcua drops
// Basic128Rsa15 when a server lists several (its comment names S7), and
// open62541 picks the matching policy in a default build, where
// Basic128Rsa15 is not compiled in; all four read from source at their
// 2026-11-18 heads and not run here.
package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gopcua/opcua/ua"
)

// rsa15URI is the user-token encryption URI of Basic128Rsa15, the legacy
// policy both arms list first - Part 7's profile table for Basic128Rsa15,
// which the tree quotes at uapolicy/policyBasic128Rsa15.go:30:
// "AsymmetricEncryptionAlgorithm – Rsa15 –
// (http://www.w3.org/2001/04/xmlenc#rsa-1_5)." A literal, never read from
// uapolicy, whose EncryptionURI() would let the library grade itself. It
// is the algorithm half of the defect the knownBug constant names: the
// URI a client that takes the first, legacy policy encrypts the password
// with, and the one the known-bug arm keys on.
const rsa15URI = "http://www.w3.org/2001/04/xmlenc#rsa-1_5"

func TestLegacyFirstUsernamePoliciesUseTheEndpointsOwn(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is
	// logged in place of a failure, and the arms below tell a pass that
	// it is time to delete it. ../README.md says when.
	const knownBug = "SecurityFromEndpoint takes the first USERNAME " +
		"UserTokenPolicy in the endpoint's list, so a non-conforming server that " +
		"lists a legacy policy first has the client encrypt its password under the " +
		"legacy policy instead of the endpoint's own or one that names none"

	t.Run("a matching policy later in the list", func(t *testing.T) {
		// The server's own list, unedited: the stimulus initEndpoints
		// builds because the harness enabled Basic128Rsa15 first.
		o, teardown := runArm(nil)
		defer teardown()
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
		// The verdict switch sits here, in the test function, per
		// ../README.md - only a test function calls t.Errorf or t.Logf,
		// a helper returns plain values - and it closes over the
		// knownBug constant, so deleting the constant stops the arms
		// that read it compiling. The shape is the one every
		// constant-carrying row in these suites uses: the defect the
		// constant names logs the known bug instead of failing, any
		// other wrong outcome fails anyway, the right outcome while the
		// constant stands is a hard error telling the changer to delete
		// it, and the constant gone, the right outcome logs the pass and
		// the wrong one fails. got and want name both fields the row
		// scores, so a reader can tell a client that picked the right
		// policy and encrypted it wrongly from one that picked the wrong
		// policy outright, and legacyPick says the recorded token carried
		// the first, legacy policy's PolicyID and its rsa-1_5 algorithm -
		// the defect itself, and the only outcome the known-bug arm
		// stands for.
		got, want, conforming, legacyPick := scoreChosenPolicy(o, "username_basic256sha256")
		switch {
		case legacyPick && knownBug != "":
			t.Logf("known bug: %s (%s, not %s)", knownBug, got, want)
		case conforming && knownBug != "":
			t.Errorf("this now passes (%s) - delete the knownBug constant, "+
				"every arm that reads it, and anything only those arms used, "+
				"in the commit that fixed it", got)
		case conforming:
			t.Logf("%s - the conforming candidate", got)
		default:
			t.Errorf("%s, not %s - the matching policy the endpoint's own "+
				"SecurityPolicy names, which the should-sentence has a conforming "+
				"server offer and a client meeting a non-conforming list use",
				got, want)
		}
	})

	t.Run("an empty-policy entry later in the list", func(t *testing.T) {
		// The other shape the should-sentence names: a legacy policy
		// first, one that names none after it. The PolicyIDs are the
		// row's own strings, so the recorded request can only name a
		// policy the row wrote out.
		o, teardown := runArm([]*ua.UserTokenPolicy{
			{
				PolicyID:          "username_legacy",
				TokenType:         ua.UserTokenTypeUserName,
				SecurityPolicyURI: ua.SecurityPolicyURIBasic128Rsa15,
			},
			{
				PolicyID:  "username_unnamed",
				TokenType: ua.UserTokenTypeUserName,
				// SecurityPolicyURI stays the empty string.
			},
		})
		defer teardown()
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
		// The same verdict switch as the first arm's, for the same
		// reasons: in the test function, closing over the knownBug
		// constant - except the failure message, which names this arm's
		// own expectation: the policy that names no SecurityPolicy, not
		// the one the endpoint's own SecurityPolicy names.
		got, want, conforming, legacyPick := scoreChosenPolicy(o, "username_unnamed")
		switch {
		case legacyPick && knownBug != "":
			t.Logf("known bug: %s (%s, not %s)", knownBug, got, want)
		case conforming && knownBug != "":
			t.Errorf("this now passes (%s) - delete the knownBug constant, "+
				"every arm that reads it, and anything only those arms used, "+
				"in the commit that fixed it", got)
		case conforming:
			t.Logf("%s - the conforming candidate", got)
		default:
			t.Errorf("%s, not %s - the policy that names no SecurityPolicy "+
				"at all, which the should-sentence lets a conforming server "+
				"offer beside its own and whose empty securityPolicyUri hands "+
				"the client the channel's own policy",
				got, want)
		}
	})
}

// armOutcome is what one arm's run left within the row's reach: the
// UserNameIdentityToken the client put in its ActivateSession request, the
// USERNAME policies the presented endpoint listed, and the reasons the
// run cannot be scored or is already a plain failure - "" when it can be
// and is not. setupWhy is the reason the run cannot be scored, and
// fromGopcua says which side caused it, per the header's split: the arm
// declines when the fixture, the environment or the row's own rewrite
// caused it, and scores gopcua's own behaviour as unscoreable when the
// library did. failWhy is a failure the arm scores outright, neither a
// decline nor unscoreable: the run reached the stimulus and the client
// did something the row can pin on it - a recorded token of the wrong
// type. A run that scored carries the token and the policies.
type armOutcome struct {
	token      *ua.UserNameIdentityToken
	policies   []*ua.UserTokenPolicy
	setupWhy   string
	fromGopcua bool
	failWhy    string
}

// runArm brings one arm up end to end: the in-tree server on an ephemeral
// port, its Basic256Sha256 SignAndEncrypt endpoint description fetched over
// the wire, the USERNAME token list the arm presents - the server's own
// when tokens is nil, the arm's rewrite of it otherwise - and a real gopcua
// client connected over the wire, activated with the harness's rowUser and
// rowPass. It returns what the arm's run left to score and a teardown for
// everything that came up; it returns plain values, and the arm turns a
// non-empty setupWhy into the decline or the unscoreable score, per
// fromGopcua, and a non-empty failWhy into the outright failure. A stage
// that fails leaves the harness partly up: teardown is non-nil for
// whatever came up, and the arm still owns calling it.
//
// The arm's precondition is checked here because both arms share it: the
// presented list must hold several USERNAME policies, the first naming a
// policy that is neither the endpoint's own nor empty - the legacy policy
// that gives the row its name - and a later one a conforming server could
// have offered, naming the endpoint's own policy or none. A list without
// that shape never presented the stimulus, and the run cannot be scored;
// the list is the in-tree server's own construction when tokens is nil, so
// the reason is scored as gopcua's behaviour then, while a rewritten list
// that does not hold the shape is the arm's own construction and declines -
// the row cannot pin its own token list on the library.
func runArm(tokens []*ua.UserTokenPolicy) (o armOutcome, teardown func()) {
	o = armOutcome{}
	teardown = func() {}

	s, serverTeardown, why := startSessionServer()
	teardown = serverTeardown
	if why != "" {
		o.setupWhy = why
		return o, teardown
	}

	ep, why := s.endpoint(ua.SecurityPolicyURIBasic256Sha256, ua.MessageSecurityModeSignAndEncrypt)
	if why != "" {
		o.setupWhy, o.fromGopcua = why, true
		return o, teardown
	}
	if tokens != nil {
		ep = withUsernameTokens(ep, tokens...)
	}
	o.policies = usernamePolicies(ep)

	if why := stimulusShapeFailed(o.policies, ep); why != "" {
		o.setupWhy, o.fromGopcua = why, tokens == nil
		return o, teardown
	}

	waitConnect, why, fromGopcua := launchClient(s, ep, rowUser, rowPass)
	if why != "" {
		o.setupWhy, o.fromGopcua = why, fromGopcua
		return o, teardown
	}
	// Connect has returned - or outlived its bound - so every request the
	// client was going to send is in; the teardown the arm registers cannot
	// close the server underneath it, per harness.go.
	waitConnect()

	// The wrapper records the request before it answers it, and Connect
	// does not return before the answer arrives, so the request is already
	// in the channel; afterConnectBound is a grace for the handoff and not
	// a second connectBound's wait for a request that can no longer arrive.
	req, why := s.awaitActivateSession(afterConnectBound)
	if why != "" {
		o.setupWhy = why
		return o, teardown
	}

	tok, why, unscoreable := usernameToken(req)
	if why != "" {
		// A token the row cannot read at all is gopcua's own behaviour and
		// scores as unscoreable; a token of another type decoded fine, and
		// the client was configured to send a UserNameIdentityToken and did
		// not, so the row scores it as the outright failure it is - the
		// split usernameToken's own comment in harness.go states.
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

// stimulusShapeFailed reports why the endpoint's USERNAME policies do not
// hold the shape the row's premise needs - several, a legacy one first, a
// conforming candidate after it - or "" if they do. The legacy policy is
// one that names a policy the endpoint does not run; the conforming
// candidate is one that names the endpoint's own or names none, the two
// shapes the should-sentence gives a conforming server.
func stimulusShapeFailed(pols []*ua.UserTokenPolicy, ep *ua.EndpointDescription) string {
	if len(pols) < 2 {
		return fmt.Sprintf("the endpoint lists %d USERNAME UserTokenPolicy, not the "+
			"several the row's premise needs", len(pols))
	}
	if pols[0].SecurityPolicyURI == "" || pols[0].SecurityPolicyURI == ep.SecurityPolicyURI {
		return fmt.Sprintf("the endpoint's first USERNAME policy is %q, not the legacy "+
			"policy - one the endpoint does not run - that the row's premise lists first",
			pols[0].SecurityPolicyURI)
	}
	for _, t := range pols[1:] {
		if t.SecurityPolicyURI == ep.SecurityPolicyURI || t.SecurityPolicyURI == "" {
			return ""
		}
	}
	return fmt.Sprintf("after its first, legacy policy %q, the endpoint lists no USERNAME "+
		"policy that names the endpoint's own %q or names none, so the conforming "+
		"candidate the row scores against is not there",
		pols[0].SecurityPolicyURI, ep.SecurityPolicyURI)
}

// scoreChosenPolicy scores one arm's recorded token against the arm's
// expected PolicyID and the channel policy's EncryptionAlgorithm. It
// returns plain values, per ../README.md's rule that only a test
// function calls t.Errorf or t.Fatalf: got is the USERNAME list the arm
// presented and the PolicyID and EncryptionAlgorithm the client sent,
// want is the pair the arm expected, conforming says whether the
// recorded token carries them, and legacyPick says whether it carries
// the first, legacy policy's PolicyID and that policy's rsa-1_5
// algorithm - exactly the defect the knownBug constant names, so the
// verdict switch can key its known-bug arm on the defect itself and
// fail every other wrong outcome while the constant stands. The subtest
// closures turn the four into the verdict - the known bug, the pass or
// the failure - and the verdict switch there reads the knownBug constant
// they close over.
func scoreChosenPolicy(o armOutcome, wantPolicyID string) (got, want string, conforming, legacyPick bool) {
	wantAlg := rsaOAEPURI
	got = fmt.Sprintf("the endpoint listed %s, and the client sent PolicyID %q and EncryptionAlgorithm %q",
		usernamePoliciesDesc(o.policies), o.token.PolicyID, o.token.EncryptionAlgorithm)
	want = fmt.Sprintf("PolicyID %q and EncryptionAlgorithm %q", wantPolicyID, wantAlg)
	conforming = o.token.PolicyID == wantPolicyID && o.token.EncryptionAlgorithm == wantAlg
	legacyPick = o.token.PolicyID == o.policies[0].PolicyID && o.token.EncryptionAlgorithm == rsa15URI
	return got, want, conforming, legacyPick
}

// usernamePoliciesDesc names the USERNAME policies a list holds, in order,
// the way the row's log and failure lines read them: each as
// policyID=securityPolicyUri, an empty securityPolicyUri shown as naming
// none.
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
