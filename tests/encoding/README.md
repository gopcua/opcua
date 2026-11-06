# Encoding tests

The shared contract for these suites is in `../README.md`; this file holds
only what is specific to encoding.

There is no scripted server here: a row drives `ua.Encode` and `ua.Decode`
directly, so what the row runs against - item 3 in `../README.md` - names
the shared helpers in `harness.go` rather than a server.

Panic recovery is this package's analogue of reconnect's decline
machinery. `encodeRecovering` calls `ua.Encode` and turns a nil
dereference into a value; `decodeVariantRecovering` does the same for
`ua.Decode` on a Variant packet, whose array length reaches
`reflect.MakeSlice` straight off the wire. A missing nil guard is a nil
dereference that takes the whole test binary with it, so one unguarded
type would hide the state of every other; the helpers turn the panic into
one case's failure instead, and only a test function turns any of it into
a failure.

A reference stack here means another stack's codec, not its client. Three
have been read. open62541's `ua_types_encoding_binary.c` and
UA-.NETStandard's `BinaryEncoder.cs` cover the encode side; the functions
read, and the line numbers, are named in the header of
`nil_builtin_encodes_as_its_null_value_test.go`. The three Variant rows
also read the decode side - open62541's same file, UA-.NETStandard's
`BinaryDecoder.cs` and `MatrixOf.cs`, and Eclipse Milo's
`OpcUaBinaryDecoder.java` - and each of those rows names the functions
and lines in its own header.

Milo is read by the Variant rows, and by
`datetime_out_of_range_saturates_test.go`, whose decode-side question its
`decodeDateTime` answers directly. On all three of the Variant rows'
packets it is the stack whose structure is closest to what those rows
require, and on one of them it is the only stack that rejects. A row that
cites it says so; a row that does not has not read it.

## The knownBug ledger, 2026-11-05

One constant stands per row below, each declared inside its test
function.

| row | closed by |
|---|---|
| `datetime_out_of_range_saturates_test.go` | nothing open closes this |
| `reference_description_with_nil_fields_encodes_test.go` | nothing open closes this |
| `variant_rejects_an_array_length_that_is_not_a_count_test.go` | [pull 892](https://github.com/gopcua/opcua/pull/892) and [pull 936](https://github.com/gopcua/opcua/pull/936) |
| `variant_rejects_dimensions_inconsistent_with_the_array_length_test.go` | nothing open closes this |
| `variant_rejects_more_dimensions_than_the_message_can_hold_test.go` | [pull 893](https://github.com/gopcua/opcua/pull/893) and [pull 936](https://github.com/gopcua/opcua/pull/936) |

`variant_rejects_dimensions_inconsistent_with_the_array_length_test.go`
names pull 892, pull 893 and pull 936 as checked and fixing none of it,
so its constant stands until one does.
