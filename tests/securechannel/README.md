# Secure channel tests

The shared contract for these suites is in `../README.md`; this file holds
only what is specific to the securechannel package.

A row here scores what the **SecureChannel** does between the transport
and the algorithms: OPC 10000-6 §6.7 and OPC 10000-4 §5.6 govern the
channel's lifecycle and the verification of the chunks it carries, and
gopcua's `uasc` package is that channel in code. The rows score four
behaviours a client sees from the wire: a security token's life measured
on the client's clock, the channel's survival of a ServiceFault, the
rejection of a chunk secured with an expired token, and the verification
of the sequence numbers the channel receives.

What the package does not score: the HEL/ACK exchange that precedes the
OPN is `../transport`'s, the algorithms inside the OPN are
`../securitypolicy`'s, and the sender direction of OPC 10000-6
§6.7.2.4 - the client's own chunks incrementing by exactly one - is
spec-clean on every tree measured, so it is not a row. The
expired-token row scores only its Basic256Sha256/Sign arm: under
SecurityPolicy None the receive path never compares the TokenId on the
wire, so an old-token chunk is indistinguishable from a well-secured one
on a fixed tree too, and a None arm cannot score the behaviour.

## The scripted peer

- **A scripted peer plays the server's half of the secure channel.** A
  plain TCP peer built on `net`, not gopcua's uacp server, listening on
  `127.0.0.1:0` - never port 4840, which other suites bind. It answers
  the HEL with the ACK, answers the OPN with a security token whose
  channel id, token id, createdAt and lifetime the row chose - and the
  client's own renewal OPN the same way - and answers the client's MSG
  requests with the response bodies and sequence numbers the row's
  script picked. It accepts connections until teardown, each on its own
  generation, so a client that drops the channel and dials again is
  served and observed on the way back. The listen, the HEL/ACK
  opening of each connection, and the split of a failed read into one
  that merely ended and one that only the client's own bytes can
  explain are the core the package shares with the transport and
  securitypolicy packages, in `../internal/scriptedpeer`. The client
  under test is the real `opcua.Client`, built by each row with the
  options its behaviour needs; the harness launches nothing, because
  a row here drives its client's requests itself and holds its handle.
- **Every number the rows score comes from the script, never from the
  library under test.** The token's ids, stamp and lifetime, the
  response bodies, the sequence numbers stamped on them, and which
  token an answer is stamped with - the current one by default, the
  one a row's script picks otherwise - are the row's own choices; the
  peer's records - accepts, OPNs answered with the generation each
  arrived on and whether each asked for a new token or renewed the
  current one, requests with the generation each arrived on, ends of
  reads, each marked when it was the read bound passing rather than a
  client ending the connection - are what the rows read back, and the
  ServiceFault row's verdict rests on those records alone.
- **Rows 1, 2 and 4 run under SecurityPolicy None**, so every chunk is
  in the clear and the harness's `noneCodec` encodes and decodes them.
  The expired-token row runs under Basic256Sha256 in mode Sign, and the
  codec for that policy - the asymmetric and symmetric signing, with
  keys minted in memory at test time, nothing on disk - lives with that
  row, because no second row calls it.
- **A wall clock bounds every read.** The peer caps each read at
  `bound` in `harness.go`; the bound is not a timing assumption - a
  correct client sends each next chunk in milliseconds - it lets a slow
  machine finish and releases a client that stopped talking. Every
  listener and connection is closed in `t.Cleanup`, so an exchange left
  blocked by the library under test is released when the test ends. The
  rows bound their own client calls with contexts and request timeouts
  the way the clauses' lifetimes demand; the expired-token row's window
  - a short token, the client's own renewal, the 25 % grace - is part
  of its stimulus and lives in its own file.

Item 3's reading here is that fixture, and it is checkable the way it is
in the other packages: `grep -c "See harness.go"
tests/securechannel/*_test.go` prints 1 for every row; the glob does not
match `harness.go`, which is the fixture itself and not a row.

## Handed to its caller

Two rows - expired_security_token_is_rejected_test.go and
received_sequence_number_is_verified_test.go - score a chunk the
client must not hand to its caller, and the phrase means one
observable: the response body arrives at the caller, or it does not.
The rejection is the body never arriving; the acceptance, the knownBug
arm, is the body arriving. On these unfixed trees the rejection shows
as the request returning its own timeout error, the timeout the row
bound it with, because a rejected chunk reaches no handler and so the
request is never answered - but that is how the rejection shows here,
not what it is: a fixed client that closes the channel on a failed
check (§6.7.7) returns the closed channel's error, an EOF, not a
timeout. So the rows key on the body and never on which error the
request returns along the way.

## When a row declines

Every row declines when the fixture fails its half of the exchange: the
peer's own failure - its encoding, a write it could not complete, the
ACK not going out - no OPN, no session request where the row's story
needs one; a write the peer could not complete records its fixture
reason beside the close record, so `setupCheck` declines it by name.
Bytes the peer's framing rejects - a bad header, an out-of-range size,
an ERR frame - and a chunk the codec cannot decode can only be the
client's own, so the harness's `setupCheck` reports them as gopcua's
behaviour, not as a decline; the same split names a connection that
merely ended - a close, a reset, a teardown - a fact the rows read in
the close records, where the read bound's own ends are marked too, so
a row can tell an idle connection the bound released from a client's
close. Two rows score what they see rather than decline on
the client's own confusion: the lifetime row's skewed arm has no decline
of its own, because what a client does with a backdated token is the
behaviour it scores; the expired-token row's scored arm scores a client
whose own behaviour leaves the arm unscoreable, and its control arm
declines, because the instrument check never ran. Each row's header
says which stages its own check covers; the check the rows share is
`setupCheck` in `harness.go`.

## The knownBug ledger

Two constants stand, in the two rows that still need one, each
declared in its row's test function, each naming the production lines
that decide the outcome and what deletes it:

- `expired_security_token_is_rejected_test.go`: issue #941; the
  expiration is keyed by the token id where every lookup is by the
  channel id; keying it by the channel id deletes it.
- `received_sequence_number_is_verified_test.go`: no pull request; the
  receive path decodes the sequence header but never compares the
  number; a fix that verifies it deletes it.

Two rows pass on main and carry no constant:
`security_token_lifetime_is_measured_from_receipt_test.go`, which PR
#911 closed by measuring the token's lifetime from receipt, and
`service_fault_leaves_the_channel_open_test.go`, which PR #904 closed
by keeping a service fault off the channel-error path.
