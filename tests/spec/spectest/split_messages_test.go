package spectest

import (
	"bytes"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/gopcua/opcua/uacp"
)

func makeMessage(t *testing.T, messageType string, chunkType byte, bodyLen int) []byte {
	t.Helper()
	h := uacp.Header{MessageType: messageType, ChunkType: chunkType, MessageSize: uint32(headerLen + bodyLen)}
	header, err := h.Encode()
	if err != nil {
		t.Fatalf("makeMessage(%q, %q, %d): encoding the header failed: %v", messageType, chunkType, bodyLen, err)
	}
	m := make([]byte, headerLen+bodyLen)
	copy(m, header)
	for i := range m[headerLen:] {
		m[headerLen+i] = byte(i)
	}
	return m
}

func randomBytes(rng *rand.Rand, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rng.Intn(256))
	}
	return b
}

func offsetInError(msg string) (int, bool) {
	at := strings.Index(msg, "offset ")
	if at < 0 {
		return 0, false
	}
	digits := msg[at+len("offset "):]
	end := 0
	for end < len(digits) && digits[end] >= '0' && digits[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	offset, err := strconv.Atoi(digits[:end])
	if err != nil {
		return 0, false
	}
	return offset, true
}

var validMessageTypes = []string{"HEL", "ACK", "ERR", "RHE", "OPN", "MSG", "CLO"}
var validChunkTypes = []byte{'F', 'C', 'A'}

func TestSplitMessagesReturnsCompleteMessagesAndRest(t *testing.T) {
	hel := makeMessage(t, "HEL", 'F', 0)
	msg := makeMessage(t, "MSG", 'F', 10)
	trailing := []byte{1, 2, 3, 4, 5}
	stream := bytes.Join([][]byte{hel, msg, trailing}, nil)
	messages, rest, err := splitMessages(stream)
	if err != nil {
		t.Fatalf("two valid messages plus 5 trailing bytes: err = %v, want nil", err)
	}
	if len(messages) != 2 {
		t.Fatalf("two valid messages plus 5 trailing bytes: got %d messages, want 2", len(messages))
	}
	if !bytes.Equal(messages[0], hel) {
		t.Errorf("two valid messages plus 5 trailing bytes: message 0 = %q, want the HEL message %q", messages[0], hel)
	}
	if !bytes.Equal(messages[1], msg) {
		t.Errorf("two valid messages plus 5 trailing bytes: message 1 = %q, want the MSG message %q", messages[1], msg)
	}
	if !bytes.Equal(rest, trailing) {
		t.Errorf("two valid messages plus 5 trailing bytes: rest = %q, want the 5 trailing bytes %q", rest, trailing)
	}
	if &messages[0][0] != &stream[0] {
		t.Errorf("two valid messages plus 5 trailing bytes: message 0 does not alias the input stream, want the returned messages to alias it")
	}
	if &rest[0] != &stream[len(stream)-len(rest)] {
		t.Errorf("two valid messages plus 5 trailing bytes: rest does not alias the tail of the input stream, want rest to alias it")
	}
}

func TestSplitMessagesMalformedHeaderNamesOffset(t *testing.T) {
	valid := makeMessage(t, "HEL", 'F', 3)
	bad := makeMessage(t, "MSG", 'F', 4)
	bad[0] = 'X'
	messages, rest, err := splitMessages(bytes.Join([][]byte{valid, bad}, nil))
	if err == nil {
		t.Fatalf("a header with message type XSG at byte 11 after one valid message: err is nil, want an error naming offset 11")
	}
	offset, ok := offsetInError(err.Error())
	if !ok || offset != 11 {
		t.Errorf("a header with message type XSG at byte 11 after one valid message: err = %q, want an error that names offset 11", err.Error())
	}
	if !strings.Contains(err.Error(), `invalid message type "XSG"`) {
		t.Errorf("a header with message type XSG at byte 11 after one valid message: err = %q, want text containing %q", err.Error(), `invalid message type "XSG"`)
	}
	if rest != nil {
		t.Errorf("a header with message type XSG at byte 11 after one valid message: rest = %q, want nil", rest)
	}
	if len(messages) != 1 || !bytes.Equal(messages[0], valid) {
		t.Errorf("a header with message type XSG at byte 11 after one valid message: got %d messages, want the 1 valid message before it", len(messages))
	}
}

func TestSplitMessagesInvalidChunkTypeNamesOffset(t *testing.T) {
	valid := makeMessage(t, "HEL", 'F', 3)
	bad := makeMessage(t, "MSG", 'X', 0)
	messages, rest, err := splitMessages(bytes.Join([][]byte{valid, bad}, nil))
	if err == nil {
		t.Fatalf("a header with chunk type X at byte 11 after one valid message: err is nil, want an error naming offset 11 and the chunk type")
	}
	offset, ok := offsetInError(err.Error())
	if !ok || offset != 11 {
		t.Errorf("a header with chunk type X at byte 11 after one valid message: err = %q, want an error that names offset 11", err.Error())
	}
	if !strings.Contains(err.Error(), "invalid chunk type 'X'") {
		t.Errorf("a header with chunk type X at byte 11 after one valid message: err = %q, want text containing \"invalid chunk type 'X'\"", err.Error())
	}
	if len(messages) != 1 || !bytes.Equal(messages[0], valid) {
		t.Errorf("a header with chunk type X at byte 11 after one valid message: got %d messages, want the 1 valid message before it", len(messages))
	}
	if rest != nil {
		t.Errorf("a header with chunk type X at byte 11 after one valid message: rest = %q, want nil", rest)
	}
}

func TestSplitMessagesSmallMessageSizeNamesOffset(t *testing.T) {
	valid := makeMessage(t, "HEL", 'F', 3)
	bad := []byte("MSGF\x07\x00\x00\x00")
	messages, rest, err := splitMessages(bytes.Join([][]byte{valid, bad}, nil))
	if err == nil {
		t.Fatalf("a header with message size 7 at byte 11 after one valid message: err is nil, want an error naming offset 11 and the message size")
	}
	offset, ok := offsetInError(err.Error())
	if !ok || offset != 11 {
		t.Errorf("a header with message size 7 at byte 11 after one valid message: err = %q, want an error that names offset 11", err.Error())
	}
	if !strings.Contains(err.Error(), "message size 7 is less than the header length") {
		t.Errorf("a header with message size 7 at byte 11 after one valid message: err = %q, want text containing \"message size 7 is less than the header length\"", err.Error())
	}
	if len(messages) != 1 || !bytes.Equal(messages[0], valid) {
		t.Errorf("a header with message size 7 at byte 11 after one valid message: got %d messages, want the 1 valid message before it", len(messages))
	}
	if rest != nil {
		t.Errorf("a header with message size 7 at byte 11 after one valid message: rest = %q, want nil", rest)
	}
}

func TestSplitMessagesAcceptsEveryTransportMessageType(t *testing.T) {
	messageTypes := []string{"HEL", "ACK", "ERR", "RHE", "OPN", "MSG", "CLO"}
	chunkTypes := []byte{'F', 'C', 'A'}
	want := make([][]byte, 0, len(messageTypes)*len(chunkTypes))
	for _, messageType := range messageTypes {
		for _, chunkType := range chunkTypes {
			want = append(want, makeMessage(t, messageType, chunkType, 2))
		}
	}
	messages, rest, err := splitMessages(bytes.Join(want, nil))
	if err != nil {
		t.Fatalf("one message per message type and chunk type: err = %v, want nil", err)
	}
	if rest != nil {
		t.Errorf("one message per message type and chunk type: rest = %q, want nil", rest)
	}
	if len(messages) != len(want) {
		t.Fatalf("one message per message type and chunk type: got %d messages, want %d", len(messages), len(want))
	}
	for i := range want {
		if !bytes.Equal(messages[i], want[i]) {
			t.Errorf("one message per message type and chunk type: message %d = %q, want %q", i, messages[i], want[i])
		}
	}
}

func TestSplitMessagesReturnsPartialBodyAsRest(t *testing.T) {
	hel := makeMessage(t, "HEL", 'F', 4)
	partialBody := []byte("MSGF\x0a\x00\x00\x00C")
	messages, rest, err := splitMessages(bytes.Join([][]byte{hel, partialBody}, nil))
	if err != nil {
		t.Fatalf("a complete message then 9 bytes of a message whose header announces 10: err = %v, want nil", err)
	}
	if len(messages) != 1 || !bytes.Equal(messages[0], hel) {
		t.Errorf("a complete message then 9 bytes of a message whose header announces 10: got %d messages, want the HEL message", len(messages))
	}
	if !bytes.Equal(rest, partialBody) {
		t.Errorf("a complete message then 9 bytes of a message whose header announces 10: rest = %q, want the 9 partial-body bytes %q", rest, partialBody)
	}
}

func TestSplitMessagesReassemblesMessagesAcrossSegments(t *testing.T) {
	rng := rand.New(rand.NewSource(20261001))
	for iteration := 0; iteration < 200; iteration++ {
		want := make([][]byte, 1+rng.Intn(10))
		for i := range want {
			want[i] = makeMessage(t,
				validMessageTypes[rng.Intn(len(validMessageTypes))],
				validChunkTypes[rng.Intn(len(validChunkTypes))],
				rng.Intn(301),
			)
		}
		stream := bytes.Join(want, nil)
		var got [][]byte
		var pending []byte
		for len(stream) > 0 {
			cut := 1 + rng.Intn(len(stream))
			input := make([]byte, 0, len(pending)+cut)
			input = append(input, pending...)
			input = append(input, stream[:cut]...)
			stream = stream[cut:]
			messages, rest, err := splitMessages(input)
			if err != nil {
				t.Fatalf("valid messages fed as random segments, iteration %d: err = %v, want nil", iteration, err)
			}
			got = append(got, messages...)
			pending = rest
		}
		if len(pending) != 0 {
			t.Fatalf("valid messages fed as random segments, iteration %d: %d bytes never became a message, want 0", iteration, len(pending))
		}
		if len(got) != len(want) {
			t.Fatalf("valid messages fed as random segments, iteration %d: got %d messages, want %d", iteration, len(got), len(want))
		}
		for i := range want {
			if !bytes.Equal(got[i], want[i]) {
				t.Fatalf("valid messages fed as random segments, iteration %d: message %d = %q, want %q", iteration, i, got[i], want[i])
			}
		}
	}
}

func TestSplitMessagesKeepsEveryByteWhenNoError(t *testing.T) {
	rng := rand.New(rand.NewSource(987654321))
	for iteration := 0; iteration < 200; iteration++ {
		stream := make([]byte, 0)
		count := rng.Intn(12)
		for i := 0; i < count; i++ {
			stream = append(stream, makeMessage(t,
				validMessageTypes[rng.Intn(len(validMessageTypes))],
				validChunkTypes[rng.Intn(len(validChunkTypes))],
				rng.Intn(301),
			)...)
		}
		switch rng.Intn(3) {
		case 0:
			if len(stream) > 0 {
				stream = stream[:rng.Intn(len(stream))]
			}
		case 1:
			stream = append(stream, randomBytes(rng, rng.Intn(8))...)
		case 2:
			stream = randomBytes(rng, rng.Intn(41))
		}
		messages, rest, err := splitMessages(stream)
		if err != nil {
			continue
		}
		joined := bytes.Join(messages, nil)
		joined = append(joined, rest...)
		if !bytes.Equal(joined, stream) {
			t.Fatalf("a no-error input of %d bytes, iteration %d: the messages and the rest join to a different byte sequence than the input", len(stream), iteration)
		}
	}
}

func FuzzSplitMessages(f *testing.F) {
	f.Add([]byte("HELF\x08\x00\x00\x00"))
	f.Add([]byte("OPNF\x0c\x00\x00\x00BBBBMSGF\x0a\x00\x00\x00CC"))
	f.Add([]byte("MSGF\x0a\x00"))
	f.Add([]byte("MSGF\x07\x00\x00\x00"))
	f.Add([]byte("MSGF\x0a\x00\x00\x00C"))
	f.Add([]byte("HELF\x08\x00\x00\x00MSGF\x07\x00\x00\x00"))
	f.Fuzz(func(t *testing.T, stream []byte) {
		messages, rest, err := splitMessages(stream)
		joined := bytes.Join(messages, nil)
		if err == nil {
			joined = append(joined, rest...)
			if !bytes.Equal(joined, stream) {
				t.Errorf("input %q with no error: the messages and the rest join to %q, want the input unchanged", stream, joined)
			}
			return
		}
		if len(joined) > len(stream) || !bytes.Equal(stream[:len(joined)], joined) {
			t.Errorf("input %q with error %v: the %d returned messages are not a prefix of the input", stream, err, len(messages))
			return
		}
		if rest != nil {
			t.Errorf("input %q with error %v: rest = %q, want nil", stream, err, rest)
			return
		}
		offset, ok := offsetInError(err.Error())
		if !ok {
			t.Errorf("input %q with error %v: the error names no byte offset", stream, err)
			return
		}
		if offset != len(joined) {
			t.Errorf("input %q with error %v: the error names offset %d, want %d, the total length of the returned messages", stream, err, offset, len(joined))
		}
	})
}
