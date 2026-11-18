# What this package does not test

Every client statement of OPC 10000-4 §7.41's `securityPolicyUri` field
that this harness can present has a row, is out of the package's scope
(the package README says which and why), or is below. The table exists so a reader can tell a blocked behaviour from one
nobody thought of.

| what is blocked | the sentence | why there is no row |
|---|---|---|
| a USERNAME UserTokenPolicy whose SecurityPolicy is ECC or RSA_DH based, on an endpoint whose SecurityMode is None | "If the SecurityMode is None, SecurityPolicies based on ECC or RSA_DH are not allowed and Clients shall not use UserTokenPolicies that require encryption with these SecurityPolicies." | no source in this tree names a single ECC or RSA_DH policy URI, and the client's only refusal is an accident of not implementing the policy at all — see below |

**The stimulus cannot be named.** The clause forbids the *family* —
"SecurityPolicies based on ECC or RSA_DH" — and a row's UserTokenPolicy
must carry a concrete `securityPolicyUri` string to be that family's
member. The URIs live in OPC 10000-7's SecurityPolicy tables, which
version 1.05 of Part 7 moved off the specification page onto the profiles
website (`../securitypolicy`'s README records the same move for the
policies its rows quote, which it takes from the v1.04 profile tables
the tree reproduces). Nothing in this tree names an ECC or RSA_DH URI:
`ua/enums.go` lists the six RSA-based URIs and None, the profile tables
reproduced in `uapolicy` hold none either, and no test, schema file or
comment in the repository holds one. A row that invented one would quote a URI no source supports, and nothing would
bind it to the family the clause forbids — a change that implemented the
real ECC policies would pass it untouched, so it would score nothing on
either side of the behaviour it stood for.

**The pass it could be keyed on is an accident, and the measurement.** On
this tree, measured, the client already refuses before sending: given a
None-mode endpoint whose USERNAME policy carries a `securityPolicyUri`
that is not in `uapolicy`'s map, `SecurityFromEndpoint` copies the URI
into the session's auth policy unconditionally (config.go:448-449),
`EncryptUserPassword` hands it to `uapolicy.Asymmetric`, and the lookup
fails (uapolicy/securitypolicy.go:32-35, "unsupported security policy"),
so `ActivateSession` returns the error (client.go:936-940), `Connect`
fails with `opcua: unsupported security policy <uri>`, and no
ActivateSession request reaches the server at all. That is one of the
two pass shapes the clause allows — refusing before sending — but it is
keyed on the map, not on the mode: measured the same way, a None-mode
endpoint whose USERNAME policy names an *implemented* RSA-based policy
(Basic256Sha256, which the clause's next sentence allows under None) has
the client encrypt and send, token and all. A row that scored the
refusal would therefore pass a client that violates the clause the
moment the ECC policy is implemented — the refusal would vanish with
the non-implementation, and no check anywhere on the path
(config.go:448-449, uasc/secure_channel_crypto.go:66-98) reads the
SecurityMode — and the row would be green on both sides of the defect
it existed to catch. What would close it: the ECC policy URIs from a
Part 7 or profiles source, read and quoted, at which point a row keyed
on the recorded request can be written exactly the way its sibling rows
are.

**What is not here for scope, not blockage.** The ISSUEDTOKEN halves of
the same §7.41 sentences are not scored: gopcua's client sends an
IssuedIdentityToken with a fixed empty EncryptionAlgorithm
(client.go:955-957) and the harness has no issuer endpoint to present,
so there is no behaviour to score yet. The trust half of the None-mode
sentence is about validating the server's certificate, not about which
policy the client uses, per the package README.
