# Session tests

The shared contract for these suites is in `../README.md`; this file holds
only what is specific to the session package.

A row here scores what the **client** does with the `securityPolicyUri` of
a USERNAME UserTokenPolicy — one clause of the standard, OPC 10000-4 §7.41
UserTokenPolicy, field securityPolicyUri (v1.05), which says what the field
is for ("The security policy to use when encrypting or signing the
UserIdentityToken when it is passed to the Server in the ActivateSession
request"), what an empty one means ("The security policy for the
SecureChannel is used if this value is null or empty"), what a conforming
server offers under a secured mode, and what a client may not use under
SecurityMode None. The conformance unit is OPC 10000-7's "Security User
Name Password Client" (User Token - User Name Password Client Facet). The
whole package is about USERNAME policies; the ISSUEDTOKEN halves of the
same sentences are not scored, and NOT-TESTED.md says what blocks them
where something does.

The observable every row scores is what reaches the server: the
UserIdentityToken of the recorded ActivateSession request, meaning its
PolicyID and the EncryptionAlgorithm it names. Connect succeeding or
failing is only a symptom, and no row keys on it — the server here checks
no user credentials at all.

## The fixture

- **The in-tree gopcua server presents the endpoint descriptions, and a
  wrapped handler records the request.** The server comes up on an
  ephemeral port — never port 4840, which other suites bind — with
  SecurityPolicy None, Basic128Rsa15 in mode SignAndEncrypt and
  Basic256Sha256 in mode SignAndEncrypt enabled, in that order, and both
  auth modes, Anonymous and UserName. `initEndpoints`
  (server/server.go:373) builds each endpoint's UserIdentityTokens from
  the enabled policies in the order they were enabled
  (server/server.go:400-435), so the harness's order is the order the
  client meets: the Basic256Sha256 endpoint lists the USERNAME policy
  `username_basic128rsa15` first and its own `username_basic256sha256`
  after it. That is the in-tree server's own construction, and the
  interop row's first arm presents it unedited — the in-tree server
  breaks §7.41's no-more-than-one sentence by construction, which is what
  makes the interop stimulus available without fixture surgery.
- **The ActivateSession handler is wrapped, and its Good answer is the
  in-tree handler's own.**
  `(*server.Server).RegisterHandler` (server/service_handlers.go:96) only
  writes a handler that is not yet registered, so a handler registered
  before `Start` is the one that stays — `initHandlers`' own registration
  runs at Start (server/server.go:250) and leaves ours. The wrapper
  records the request and answers it with the in-tree handler's own
  answer to a request whose session it finds and whose client signature
  verifies: a Good ServiceResult and a fresh 32-byte
  ServerNonce (server/session_service.go:114-128). It reproduces nothing
  else of that handler — the in-tree one also resolves the request's
  AuthenticationToken to a session, refusing when there is none, and
  runs `sc.VerifySessionSignature` on the client's signature, refusing
  when that fails (server/session_service.go:101-112), and the wrapper
  does neither. It checks nothing about the request's UserIdentityToken,
  the way the in-tree handler does not (server/session_service.go
  ActivateSession reads the UserIdentityToken and does nothing with it)
  — the request is the evidence, and the rows hold the verdict.
- **The client under test is the real `opcua.Client`, driven the way an
  application drives it**: the row fetches the server's own endpoint
  descriptions with a real `GetEndpoints`, picks the one it needs, and
  hands it to `SecurityFromEndpoint(ep, ua.UserTokenTypeUserName)` with
  `AuthUsername` — the selection of which UserTokenPolicy to use and the
  encryption of the password are the library's own code
  (config.go:429-453, uasc/secure_channel_crypto.go:66-98), not the
  row's. Where a row needs an endpoint list the in-tree server's own
  construction cannot produce — a USERNAME policy with an empty
  `securityPolicyUri`, a legacy-then-unnamed list — it rewrites the
  UserIdentityTokens of the real description and nothing else, the
  narrowest surgery that presents the list; every other field of the
  description (its URL, its certificate, its security policy and mode)
  stays the real server's own. The row's headers say which arms rewrite
  and which present the server's own list.
- **Keys and certificates are minted in memory at test time**, 2048-bit
  RSA, self-signed X.509, one pair for the server and one per client;
  nothing is written to disk and no key material is committed. 2048 is
  not a free choice: Basic128Rsa15's profile caps the asymmetric key at
  2048 bits (uapolicy/policyBasic128Rsa15.go:73-74) and Basic256Sha256
  requires at least that (uapolicy/policyBasic256Sha256.go:86), and on
  a tree where the client picks the legacy policy — the interop row's
  known bug — it encrypts the password under Basic128Rsa15 with the
  server's very certificate, so a longer key would turn that row's red
  into an unscoreable run.
- **A wall clock bounds every stage**: a Connect, a GetEndpoints, and —
  only after Connect has returned, because the wrapper records a request
  before it answers it and Connect does not return before the answer
  arrives — a short grace (`afterConnectBound`, not a second
  `connectBound`) for the handoff of the recorded request. The bound is
  not a timing assumption: a correct pair finishes each stage in well
  under a second, so the bound lets a slow machine finish rather than
  waits out a broken one. A row waits for its client's Connect to return
  before tearing the server down: the recorded request arrives
  mid-Connect, and a teardown that closes the server underneath a Connect
  still in flight fails the client's own reads — the pair wedge the
  teardown then waits out — for the fixture's timing and not for anything
  the row scores.

Item 3's reading here is that fixture, and it is checkable the way it is
in the other packages: `grep -c "See harness.go" tests/session/*_test.go`
prints 1 for every row; the glob does not match `harness.go`, which is
the fixture itself and not a row.

## Where a row's evidence comes from

`../README.md`'s rule says a verdict rests on bytes from outside gopcua.
The rows here read a request that gopcua's server half decoded, so the
recorded token crossed one gopcua codec between the client and the row;
the same caveat the server package states applies, and the rows answer it
the same two ways:

- **The values that decide a verdict are literals in the row, never read
  from the library under test.** The expected EncryptionAlgorithm URIs
  come from Part 7's profile tables as the tree reproduces them
  (uapolicy/policyBasic256Sha256.go:38-42: "The URI is
  http://www.w3.org/2001/04/xmlenc#rsa-oaep."), and the expected PolicyIDs
  are strings the row itself wrote into the endpoint description — so a
  recorded request can only name a policy the row presented. Reading
  `uapolicy`'s EncryptionURI() would let the library grade itself.
- **The byte-level row decrypts with the standard library.**
  `empty_token_policy_means_the_channels_policy_test.go` decrypts the
  recorded Password with `crypto/rsa.DecryptOAEP` under SHA-1 and the
  server's minted key — never with gopcua code — and requires the
  plaintext to carry the row's own password literal, so a client that
  names the channel policy's algorithm and encrypts with another one
  fails. The interop row rests on that proof and names it, per
  `../README.md`'s rule that a row may lean on another row that holds the
  bytes.

What the fixture deliberately does not add is a second stack's client:
like the server package's rows, both halves here are gopcua, and a
symmetric codec bug that survived the stdlib decryption would be
invisible to these rows. Four reference stacks' clients were read from
source for both rows (UA-.NETStandard, Eclipse Milo, open62541,
node-opcua); each row's header says what they do, including the one that
fails the legacy-first row.

## When a row declines

Every row declines when the fixture fails its half: no free port, a key
that would not mint — the server's or the client's, both minted by the
harness — a server that would not start, no ActivateSession request
within the harness's `afterConnectBound` grace — a client that never
sent one and a server that never answered its half land in the same
reason, and a red the row cannot pin on the library is not a score. A
stage where the library is its own fixture — a GetEndpoints that would
not complete, a client `opcua.NewClient` would not build, the server's
own token list when it does not hold the shape a row's premise needs,
a recorded token the row cannot read — is scored as gopcua's own
behaviour made the run unscoreable, with `t.Errorf`, per the server
package's precedent that both halves are the library under test. Two
reasons sit outside that split: a token list the row itself rewrote that
does not hold the shape its premise needs is the row's own construction
and declines, and a recorded token of the wrong type is a failure the row
scores outright — the client was configured to send a
UserNameIdentityToken and did not, which is not a reason the run cannot
be scored.

## The knownBug ledger, 2026-11-18

One constant stands, declared in
`legacy_first_username_policies_use_the_endpoints_own_test.go`'s test
function.

| row | state |
|---|---|
| `empty_token_policy_means_the_channels_policy_test.go` | no constant; gopcua passes it as it stands (measured 2026-11-18 on this tree, 0.2s), and the recorded request carried the offered PolicyID, the rsa-oaep EncryptionAlgorithm and a password that decrypted to the row's own. Two library paths produce that observable together: `SecurityFromEndpoint` reads the empty `securityPolicyUri` and sets the session's auth policy to the endpoint's own (config.go:448-452), and `EncryptUserPassword` falls back to the channel's own policy when the auth policy it is handed is empty (uasc/secure_channel_crypto.go:67-70) — the row scores what reaches the server, so it stays green while either path runs. The guard, measured as four one-line mutants on a scratch copy of this tree: setting the else-branch's assignment to `ua.SecurityPolicyURINone` sends the password in the clear and turns the row red — both the algorithm arm and the decryption arm fire, the only mutant that turns it red in a scored way; deleting the else-branch alone leaves the row green, because `EncryptUserPassword`'s fallback hands the channel's own policy to the empty auth policy the deletion leaves; deleting that fallback alone leaves the row green too, because the else-branch has already set the auth policy, so the fallback never runs; deleting both has the client fail to encrypt before it sends anything — `uapolicy.Asymmetric("")` errors, `Connect` fails, no ActivateSession arrives — and the row declines (`could not set the test up: no ActivateSession request arrived`) rather than scoring. |
| `legacy_first_username_policies_use_the_endpoints_own_test.go` | `knownBug` stands: `SecurityFromEndpoint` walks the endpoint's UserIdentityTokens in order and returns on the first whose TokenType matches (config.go:429-453), so both arms — the server's own legacy-first list, and the rewritten legacy-then-unnamed one — sent the legacy policy's PolicyID and encrypted the password with `http://www.w3.org/2001/04/xmlenc#rsa-1_5`, Basic128Rsa15's user-token URI (measured 2026-11-18 on this tree). The row keys its known-bug arm on exactly the defect the constant names — a request that arrived carrying the first, legacy policy's PolicyID and that policy's rsa-1_5 algorithm — and its deletion arm on one that named the right one; any other wrong outcome, the right PolicyID with the wrong algorithm above all, is a hard failure even while the constant stands; measured, the one-line skip of the non-matching policy in that same loop turns both arms green and fires the deletion arm. |

The constant is expected to be closed by a change to `SecurityFromEndpoint`'s
selection: skip USERNAME policies that name a policy the endpoint does not
run, and take the first that names the endpoint's own or names none.

## What the package deliberately does not score

Two more sentences of the same §7.41 field have no row here, for reasons
that are scope and not fixture gaps:

- "the Client shall only use a ServerCertificate which it trusts to
  encrypt UserIdentityTokens with tokenType USERNAME or ISSUEDTOKEN" —
  this is about validating the server's certificate, not about which
  policy the client uses, so it is outside this package's question.
- "When a UserTokenPolicy is returned in an EndpointDescription all of
  the information needed to use that UserTokenPolicy shall be in the
  EndpointDescription" — a shall on the server's endpoint description,
  not a client behaviour with `securityPolicyUri`, so it is not this
  package's to score; the row that would score it belongs with the
  server's endpoint construction.

NOT-TESTED.md holds the sentence that has no row for a reason the fixture
and the tree cannot cure — the ECC and RSA_DH prohibition under
SecurityMode None — and its last section says why the ISSUEDTOKEN halves
of the same sentences are not scored either.
