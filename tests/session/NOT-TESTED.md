# What this package does not test

Every client statement of OPC 10000-4 §7.41's `securityPolicyUri` field
that this harness can present has a row, is out of scope (the package
README says why), or is listed below. The table lets a reader tell a
blocked behaviour from one nobody thought of.

| what is blocked | the sentence | why there is no row |
|---|---|---|
| a USERNAME UserTokenPolicy whose SecurityPolicy is ECC or RSA_DH based, on an endpoint whose SecurityMode is None | "If the SecurityMode is None, SecurityPolicies based on ECC or RSA_DH are not allowed and Clients shall not use UserTokenPolicies that require encryption with these SecurityPolicies." | no source in this tree names an ECC or RSA_DH policy URI, and the client refuses today only because it does not implement them; see below |

**The stimulus cannot be named.** A row needs a concrete ECC or RSA_DH
`securityPolicyUri`. Those URIs are in OPC 10000-7's SecurityPolicy
tables, which v1.05 moved to the profiles website (`../securitypolicy`'s
README records the same move). Nothing in this tree names one: `ua/enums.go`
lists None and the six RSA-based URIs, and neither `uapolicy`'s profile
tables nor any test, schema or comment holds one. A row with an invented
URI would quote no source and would still pass once real ECC policies were
implemented.

**Today's refusal is an accident.** Measured on this tree: for a None-mode
endpoint whose USERNAME policy names a URI `uapolicy` does not know,
`SecurityFromEndpoint` copies the URI into the auth policy
(config.go:465-466), `uapolicy.Asymmetric` rejects it
(uapolicy/securitypolicy.go:32-35), and `Connect` fails with
`opcua: unsupported security policy <uri>` before any ActivateSession is
sent. That refusal comes from the missing policy, not from the
SecurityMode: a None-mode endpoint whose USERNAME policy names
Basic256Sha256 has the client encrypt and send. Nothing on the path
(config.go:465-466, uasc/secure_channel_crypto.go:66-98) reads the
SecurityMode, so a row that scored the refusal would stay green after ECC
support landed without the check. What would close it: ECC policy URIs
quoted from a Part 7 or profiles source, so a row can key on the recorded
request like its siblings.

**Out of scope, not blocked.** The ISSUEDTOKEN halves of the same §7.41
sentences are not scored: gopcua's client sends an IssuedIdentityToken
with a fixed empty EncryptionAlgorithm (client.go:955-957), and the harness
has no issuer endpoint to present. The certificate-trust half of the
None-mode sentence is about validating the server's certificate, per the
package README.
