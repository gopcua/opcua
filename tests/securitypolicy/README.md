# Security policy tests

The shared contract for these suites is in `../README.md`; this file holds
only what is specific to the securitypolicy package.

A row here scores what a **SecurityPolicy** hands the secure channel when
the channel opens. OPC 10000-6 §6.1 names the SecurityPolicy as a concept
of its own - the set of algorithms a SecureChannel is opened under - and
gopcua's `uapolicy` package is that concept in code. The rows score the
two things those algorithms put inside the OpenSecureChannel request: the
plaintext block size the asymmetric encryption fills, and the client
nonce. The channel lifecycle - security tokens, their renewal - is not
scored here.

## The scripted peer and the minted keys

- **A scripted peer plays the server's half of the opening exchange.** A
  plain TCP peer built on `net`, not gopcua's uacp server, listening on
  `127.0.0.1:0` - never port 4840, which other suites bind. It accepts
  one connection, reads the HEL, answers the ACK (protocol version 0,
  65535-byte buffers in both directions, no other limits), reads the next
  frame - the OPN the rows score - records its bytes, and closes the
  connection without ever answering it. The client under test is the real
  `opcua.Client`, built by the shared `launchClient` with
  `AutoReconnect(false)` and run under a context bounded by the deadline;
  its `Connect` is expected to fail once the peer closes, and no row
  asserts its error. A row that needs a second recorded OPN - the nonce
  row and its two connections - runs the fixture twice. Bytes the peer's
  uacp framing rejects - a bad header, an out-of-range size, an ERR
  frame - can only be the client's own, so the setup check reports them
  as gopcua's behaviour, not as a decline; an exchange that ends before
  the OPN - the client closing, going away - is a
  decline the check reports at once, without waiting out its bound.
- **Keys and certificates are minted in memory at test time**: 4096-bit
  RSA for both the client and the peer, self-signed X.509, one pair for
  each; the client key is 4096 bits so its 512-byte signature makes the
  OPN span at least two blocks, and the block row scores every block.
  Nothing is written to disk and no key material is
  committed. The None row mints none: under None the client sends no
  certificate.
- **A wall clock frames the opening exchange.** A `Connect` that never
  returns cannot be observed except by outliving a bound, so the peer's
  setup check and the client's context share one (`handshake` in
  `harness.go`). The bound is not a timing assumption: a correct client
  sends its HEL and its OPN in well under a second, so the bound exists
  to let a slow machine finish, not to wait out a broken one. Every
  listener and connection is closed in `t.Cleanup`, so a `Connect` left
  blocked by the library under test is released when the test ends.

Item 3's reading here is that fixture, and it is checkable the way it is
in the other packages: `grep -c "See harness.go"
tests/securitypolicy/*_test.go` prints 1 for every row; the glob does not
match `harness.go`, which is the fixture itself and not a row.

## Why the verdicts rest outside gopcua

The peer uses gopcua's `uacp` types to frame the ACK; the rows use
`uasc` to decode the recorded frame and, in the nonce row, its request
body with `ua.DecodeService` - which is not the library grading itself,
because those types only move bytes the rows never score: the nonce
verdict rests on `len` and `bytes.Equal`.
The verdicts rest on the standard library and on the clauses: every
RSA-OAEP block is decrypted with `crypto/rsa.DecryptOAEP` and SHA-256
under the peer's minted key, never with gopcua code, and the expected
block size is a literal computed in the row from RFC 3447's arithmetic and
the minted key's size - never read from `uapolicy`, whose constant is the
known bug the block row guards against. The nonce row reads the nonce
bytes as they arrive on the wire, not through `uapolicy.NonceLength()`.

## Why Part 7 v1.04 is quoted

OPC 10000-7 v1.04 (2017-11-01) is the last version of Part 7 whose
SecurityPolicy tables are in the specification text: version 1.05 moved
them to the profiles website. The rows quote v1.04 because its Table 186
is what ties the policy URI the client names to the algorithms the rows
score.

## When a row declines

- `open_secure_channel_fills_each_rsa_oaep_block_test.go` declines
  when no OPN frame arrives, whether the bound passes or the peer's
  exchange ends first. An OPN whose asymmetric
  security header names a policy other than Aes256_Sha256_RsaPss, or
  that does not decode as a message chunk, can only be the client's
  bytes - and so are a frame off script or bytes the peer's uacp
  framing rejects, which the harness itself flags - so
  the row scores it as gopcua's own behaviour made this run
  unscoreable instead.
- `client_nonce_under_security_policy_none_is_null_or_usable_test.go`
  declines when either of its two connections' OPN did not arrive,
  whether the bound passed or the peer's exchange ended first.

## The knownBug ledger

One constant stands, declared in
`open_secure_channel_fills_each_rsa_oaep_block_test.go`'s test function:
`uapolicy` reserves 64 bytes for each SHA-256 hash where RFC 3447 §7.1.1
reserves 32 (`RSAOAEPMinPaddingSHA256 = (2 * 64) + 2`, declared in
uapolicy/crypto_rsaoaep.go and used twice: as the split size in
RSAOAEP.Encrypt and as the Aes256_Sha256_RsaPss policy's plaintext block
size), so, with the 4096-bit receiver the row mints, every
RSA-OAEP-SHA2-256 block carries 382 plaintext
bytes where the arithmetic allows 446. The nonce row carries no
constant: gopcua passes it as it stands. Its floor arm rests on
OPC 10000-4 §5.6.2.3 - a conformant server may reject a sub-32-byte
nonce - with OPC 10000-6 §6.7.4's null as the recommended shape.
