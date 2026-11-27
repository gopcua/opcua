# Session tests

The shared contract for these suites is in `../README.md`; this file holds
only what is specific to the session package.

Two rows score what the **client** does with the `securityPolicyUri` of a
USERNAME UserTokenPolicy, as OPC 10000-4 §7.41 (v1.05) defines it: "The
security policy to use when encrypting or signing the UserIdentityToken
when it is passed to the Server in the ActivateSession request [...] The
security policy for the SecureChannel is used if this value is null or
empty." The conformance unit is OPC 10000-7's "Security User Name Password
Client". The ISSUEDTOKEN halves of the same sentences are not scored;
NOT-TESTED.md says why. A third row, `unverified_server_signature_fails_connect_test.go`,
scores the handshake those rows ride on: OPC 10000-4 §6.1.8 (v1.05),
"The ClientSignature and UserTokenSignature shall not be computed until
the ServerSignature is validated."

Each of the two UserTokenPolicy rows scores the UserIdentityToken of the
ActivateSession request the server receives: its PolicyID and the
EncryptionAlgorithm it names. Neither keys on whether Connect succeeded,
because the server checks no credentials. The ServerSignature row keys on
Connect failing and on no ActivateSession request arriving at all.

## The fixture

- **The server is the in-tree gopcua server**, on an ephemeral port. It
  enables None, Basic128Rsa15 SignAndEncrypt and Basic256Sha256
  SignAndEncrypt, in that order, with the Anonymous and UserName auth
  modes. `initEndpoints` lists USERNAME policies in the order the security
  policies were enabled (server/server.go:400-435), so the Basic256Sha256
  endpoint lists `username_basic128rsa15` before its own
  `username_basic256sha256`. The in-tree server therefore breaks §7.41's
  "no more than one USERNAME UserTokenPolicy" on its own, which gives the
  legacy-first row its first arm without editing anything.
- **The ActivateSession handler is replaced.** A handler registered before
  `Start` is kept (server/service_handlers.go:96). The replacement records
  the request and answers Good with a fresh ServerNonce. Unlike the in-tree
  handler, it does not look up the session or verify the client signature.
  The ServerSignature row replaces the CreateSession handler the same way,
  through a hook `startSessionServer` runs before `Start`; its replacement
  answers with the signature the channel would sign, one byte flipped.
- **The client is the real `opcua.Client`**, configured the way an
  application configures it: GetEndpoints, then
  `SecurityFromEndpoint(ep, ua.UserTokenTypeUserName)` and `AuthUsername`.
  Choosing the policy and encrypting the password is library code
  (config.go:436-470, uasc/secure_channel_crypto.go:66-98). A row that
  needs a list the server cannot build replaces only the endpoint's
  UserIdentityTokens, and its header says so.
- **Keys are 2048-bit RSA, minted in memory** for the server and each
  client. That is the only size both Basic128Rsa15 (at most 2048) and
  Basic256Sha256 (at least 2048) accept, and the server's one certificate
  serves both of its secured endpoints.
- **Each stage has a bound**: `connectBound` for GetEndpoints and Connect,
  and the shorter `afterConnectBound` for the recorded request once
  Connect has returned. A row waits for Connect to return before tearing
  the server down, so the teardown does not break a Connect still in
  flight.

Every row names this fixture: `grep -c "See harness.go"
tests/session/*_test.go` prints 1 for each. `harness.go` is the fixture, not
a row.

## Where a row's evidence comes from

The rows read a request that gopcua's server decoded, so the token passed
through gopcua's codec on its way to the row. As in the server package, two
things keep the verdict independent of it:

- **The expected values are literals in the row.** The EncryptionAlgorithm
  URIs come from Part 7's profile tables as the tree quotes them
  (uapolicy/policyBasic256Sha256.go:38-42), and the PolicyIDs are strings
  the row wrote into the endpoint description. Reading `uapolicy`'s
  `EncryptionURI()` would let the library grade itself.
- **One row decrypts with the standard library.**
  `empty_token_policy_means_the_channels_policy_test.go` decrypts the
  Password with `crypto/rsa.DecryptOAEP` and the server's key and requires
  the plaintext to carry the row's password. A client that names one
  algorithm and encrypts with another fails. The legacy-first row relies
  on this proof and names it, as `../README.md` allows.

Both halves are still gopcua, so a symmetric codec bug that survives the
decryption would go unnoticed. UA-.NETStandard, Eclipse Milo, open62541 and
node-opcua were read from source for both rows; each row's header says what
they do.

## When a row declines

A row declines when the fixture fails: no free port, a key that would not
mint, a server that would not start, a GetEndpoints that runs past
`connectBound`, or no ActivateSession request within `afterConnectBound`
once Connect has returned. A run past a bound is a decline because a
loaded machine can cause it, as `../README.md` says. Connect runs under a
context bounded by `connectBound`, so a slow Connect is cut off and lands
in the last case. That case covers both a client that never sent the
request and a server that never received it, so the row cannot blame the
library.

A failure in gopcua's own code is scored with `t.Errorf` as "gopcua's own
behaviour made this run unscoreable", because both halves are under test:
a GetEndpoints that fails within its bound, a client `opcua.NewClient`
will not build, a server-built token list without the shape the row
needs, or a recorded token the row cannot read. Two cases differ. A token
list the row rewrote itself without the needed shape is the row's mistake
and declines. A token of the wrong type is an ordinary failure, because
the client was configured to send a UserNameIdentityToken.

## The knownBug ledger, 2026-11-18

No constant stands.

| row | state |
|---|---|
| `empty_token_policy_means_the_channels_policy_test.go` | No constant; gopcua passes (measured 2026-11-18). Two library paths fall back to the channel's policy: `SecurityFromEndpoint`'s else-branch (config.go:465-469) and `EncryptUserPassword` (uasc/secure_channel_crypto.go:67-70). Measured mutants: setting the else-branch to `ua.SecurityPolicyURINone` sends the password in the clear and turns the row red. Deleting either fallback alone leaves it green, because the other one covers. Deleting both makes `uapolicy.Asymmetric("")` fail before anything is sent, and the row declines. |
| `legacy_first_username_policies_use_the_endpoints_own_test.go` | Constant deleted when the suite was rebased onto #914; gopcua passes (measured 2026-11-18). `SecurityFromEndpoint` takes the first USERNAME policy that names the endpoint's own SecurityPolicy or none (config.go:436-447). Before #914 it took the first policy whose TokenType matched, and both arms sent the legacy PolicyID with `rsa-1_5`. A half fix that sends the right PolicyID under the legacy algorithm fails both arms (measured). |
| `unverified_server_signature_fails_connect_test.go` | No constant; gopcua passes (measured 2026-11-26). CreateSession returns the verification error and Connect hands it back (client.go:860-866). Before the fix, the error was only logged, CreateSession returned no session, and Connect dereferenced the nil session and crashed; reverting the fix fails the row with that crash (measured). |

## What the package deliberately does not score

Two more sentences of §7.41's field are out of scope:

- "the Client shall only use a ServerCertificate which it trusts to
  encrypt UserIdentityTokens with tokenType USERNAME or ISSUEDTOKEN" is
  about validating the server's certificate, not about choosing a policy.
- "When a UserTokenPolicy is returned in an EndpointDescription all of
  the information needed to use that UserTokenPolicy shall be in the
  EndpointDescription" binds the server's endpoint construction, not the
  client.

NOT-TESTED.md holds the sentence that has no row because of the tree
itself: the ECC and RSA_DH prohibition under SecurityMode None.
