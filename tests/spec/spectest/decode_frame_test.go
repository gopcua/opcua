package spectest

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

func newReadRequest(requestID uint32, node *ua.NodeID) *ua.ReadRequest {
	return &ua.ReadRequest{
		RequestHeader: &ua.RequestHeader{
			AuthenticationToken: ua.NewTwoByteNodeID(0),
			Timestamp:           time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
			RequestHandle:       requestID,
			TimeoutHint:         5000,
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
		NodesToRead:        []*ua.ReadValueID{{NodeID: node, AttributeID: ua.AttributeIDValue, DataEncoding: &ua.QualifiedName{}}},
	}
}

func encodeServiceChunks(t *testing.T, requestID uint32, maxBodySize uint32, service interface{}) [][]byte {
	t.Helper()
	typeID := ua.ServiceTypeID(service)
	if typeID == 0 {
		t.Fatalf("encoding request %d: ua.ServiceTypeID returned 0 for %T, want its registered type id", requestID, service)
	}
	message := &uasc.Message{
		MessageHeader: &uasc.MessageHeader{
			Header:                  uasc.NewHeader(uasc.MessageTypeMessage, uacp.ChunkTypeFinal, 0),
			SymmetricSecurityHeader: uasc.NewSymmetricSecurityHeader(0),
			SequenceHeader:          uasc.NewSequenceHeader(1, requestID),
		},
		TypeID:  ua.NewFourByteExpandedNodeID(0, typeID),
		Service: service,
	}
	wire, err := message.EncodeChunks(maxBodySize)
	if err != nil {
		t.Fatalf("encoding request %d: uasc.Message.EncodeChunks failed: %v", requestID, err)
	}
	if len(wire) == 0 {
		t.Fatalf("encoding request %d: uasc.Message.EncodeChunks returned no chunks", requestID)
	}
	return wire
}

func chunkFrames(t *testing.T, wire [][]byte) []*chunkFrame {
	t.Helper()
	frames := make([]*chunkFrame, 0, len(wire))
	for i, oneWire := range wire {
		frames = append(frames, decodeChunk(t, i, oneWire))
	}
	return frames
}

func decodeChunk(t *testing.T, index int, wire []byte) *chunkFrame {
	t.Helper()
	messages, rest, err := splitMessages(wire)
	if err != nil {
		t.Fatalf("decoding wire chunk %d: splitMessages failed: %v", index, err)
	}
	if len(messages) != 1 || len(rest) != 0 {
		t.Fatalf("decoding wire chunk %d: splitMessages returned %d messages and %d rest bytes, want 1 message and no rest", index, len(messages), len(rest))
	}
	decoded, err := decodeFrame(messages[0])
	if err != nil {
		t.Fatalf("decoding wire chunk %d: decodeFrame failed: %v", index, err)
	}
	frame, ok := decoded.(*chunkFrame)
	if !ok {
		t.Fatalf("decoding wire chunk %d: decodeFrame returned a %T, want a *chunkFrame", index, decoded)
	}
	return frame
}

func messageChunk(chunkType byte, requestID uint32, body []byte) ([]byte, error) {
	header := uasc.NewHeader(uasc.MessageTypeMessage, chunkType, 0)
	header.MessageSize = uint32(24 + len(body))
	buffer := ua.NewBuffer(nil)
	buffer.WriteStruct(header)
	buffer.WriteStruct(uasc.NewSymmetricSecurityHeader(0))
	buffer.WriteStruct(uasc.NewSequenceHeader(1, requestID))
	buffer.Write(body)
	return buffer.Bytes(), buffer.Error()
}

func transportMessage(messageType string, body []byte) ([]byte, error) {
	header := uacp.Header{MessageType: messageType, ChunkType: uacp.ChunkTypeFinal, MessageSize: uint32(headerLen + len(body))}
	wire, err := header.Encode()
	if err != nil {
		return nil, err
	}
	return append(wire, body...), nil
}

func TestDecodeFrameReassemblesOneFinalChunkIntoReadRequest(t *testing.T) {
	const requestID = 42
	node := ua.NewStringNodeID(1, "spectest.read")
	wire := encodeServiceChunks(t, requestID, math.MaxUint32, newReadRequest(requestID, node))
	if len(wire) != 1 {
		t.Fatalf("a ReadRequest encoded as one final chunk: got %d chunks, want 1", len(wire))
	}
	frame := chunkFrames(t, wire)[0]
	if frame.chunkType != uacp.ChunkTypeFinal {
		t.Errorf("a ReadRequest encoded as one final chunk: the chunk frame has chunk type %q, want %q (final)", frame.chunkType, uacp.ChunkTypeFinal)
	}
	if frame.requestID != requestID {
		t.Errorf("a ReadRequest encoded as one final chunk: the chunk frame has request id %d, want %d", frame.requestID, requestID)
	}
	var reassembly reassembleChunks
	assembled, complete, err := reassembly.add(frame)
	if err != nil {
		t.Fatalf("a ReadRequest encoded as one final chunk: feeding the final chunk to reassembleChunks failed: %v", err)
	}
	if !complete {
		t.Fatalf("a ReadRequest encoded as one final chunk: feeding the final chunk to reassembleChunks returned no complete message, want the complete ReadRequest")
	}
	message, ok := assembled.(*completeMessage)
	if !ok {
		t.Fatalf("a ReadRequest encoded as one final chunk: reassembleChunks returned a %T, want a *completeMessage", assembled)
	}
	if message.requestID != requestID {
		t.Errorf("a ReadRequest encoded as one final chunk: the reassembled message has request id %d, want %d", message.requestID, requestID)
	}
	read, ok := message.service.(*ua.ReadRequest)
	if !ok {
		t.Fatalf("a ReadRequest encoded as one final chunk: the reassembled service is a %T, want a *ua.ReadRequest", message.service)
	}
	if len(read.NodesToRead) != 1 {
		t.Errorf("a ReadRequest encoded as one final chunk: the reassembled ReadRequest has %d NodesToRead, want 1", len(read.NodesToRead))
	}
	if read.NodesToRead[0].NodeID.String() != node.String() {
		t.Errorf("a ReadRequest encoded as one final chunk: NodesToRead[0].NodeID = %v, want the encoded node %v", read.NodesToRead[0].NodeID, node)
	}
}

func TestDecodeFrameDecodesOpenAndCloseChunks(t *testing.T) {
	const requestID = 5
	node := ua.NewStringNodeID(1, "spectest.opn")
	typeID := ua.ServiceTypeID(&ua.ReadRequest{})
	if typeID == 0 {
		t.Fatalf("an OPN or CLO chunk: ua.ServiceTypeID returned 0 for *ua.ReadRequest, want its registered type id")
	}
	expectedBody := ua.NewBuffer(nil)
	expectedBody.WriteStruct(ua.NewFourByteExpandedNodeID(0, typeID))
	expectedBody.WriteStruct(newReadRequest(requestID, node))
	if expectedBody.Error() != nil {
		t.Fatalf("an OPN or CLO chunk: encoding the expected chunk body failed: %v", expectedBody.Error())
	}
	cases := []struct {
		name    string
		message *uasc.Message
	}{
		{
			name: "an OPN chunk with a SecurityPolicy None asymmetric security header",
			message: &uasc.Message{
				MessageHeader: &uasc.MessageHeader{
					Header:                   uasc.NewHeader(uasc.MessageTypeOpenSecureChannel, uacp.ChunkTypeFinal, 0),
					AsymmetricSecurityHeader: uasc.NewAsymmetricSecurityHeader(ua.SecurityPolicyURINone, nil, nil),
					SequenceHeader:           uasc.NewSequenceHeader(1, requestID),
				},
				TypeID:  ua.NewFourByteExpandedNodeID(0, typeID),
				Service: newReadRequest(requestID, node),
			},
		},
		{
			name: "a CLO chunk",
			message: &uasc.Message{
				MessageHeader: &uasc.MessageHeader{
					Header:                  uasc.NewHeader(uasc.MessageTypeCloseSecureChannel, uacp.ChunkTypeFinal, 0),
					SymmetricSecurityHeader: uasc.NewSymmetricSecurityHeader(0),
					SequenceHeader:          uasc.NewSequenceHeader(1, requestID),
				},
				TypeID:  ua.NewFourByteExpandedNodeID(0, typeID),
				Service: newReadRequest(requestID, node),
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			wire, err := testCase.message.EncodeChunks(math.MaxUint32)
			if err != nil {
				t.Fatalf("%s: uasc.Message.EncodeChunks failed: %v", testCase.name, err)
			}
			if len(wire) != 1 {
				t.Fatalf("%s: got %d chunks, want 1", testCase.name, len(wire))
			}
			frame := chunkFrames(t, wire)[0]
			if frame.chunkType != uacp.ChunkTypeFinal {
				t.Errorf("%s: the chunk frame has chunk type %q, want %q (final)", testCase.name, frame.chunkType, uacp.ChunkTypeFinal)
			}
			if frame.requestID != requestID {
				t.Errorf("%s: the chunk frame has request id %d, want %d", testCase.name, frame.requestID, requestID)
			}
			if !bytes.Equal(frame.body, expectedBody.Bytes()) {
				t.Errorf("%s: the chunk body starts with % x, want the service body after the sequence header % x", testCase.name, frame.body, expectedBody.Bytes())
			}
		})
	}
}

func TestReassembleChunksReassemblesAMultiChunkWriteRequest(t *testing.T) {
	const requestID = 7
	writeRequest := &ua.WriteRequest{
		RequestHeader: &ua.RequestHeader{
			AuthenticationToken: ua.NewTwoByteNodeID(0),
			Timestamp:           time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
			RequestHandle:       requestID,
			TimeoutHint:         5000,
		},
	}
	const writeValueCount = 4
	for i := 0; i < writeValueCount; i++ {
		variant, err := ua.NewVariant(fmt.Sprintf("spectest write value %d", i))
		if err != nil {
			t.Fatalf("a WriteRequest encoded into several chunks: building variant %d failed: %v", i, err)
		}
		writeRequest.NodesToWrite = append(writeRequest.NodesToWrite, &ua.WriteValue{
			NodeID:      ua.NewStringNodeID(2, fmt.Sprintf("spectest.write.%d", i)),
			AttributeID: ua.AttributeIDValue,
			Value:       &ua.DataValue{EncodingMask: ua.DataValueValue, Value: variant},
		})
	}
	wire := encodeServiceChunks(t, requestID, 64, writeRequest)
	if len(wire) < 3 {
		t.Fatalf("a WriteRequest encoded into several chunks: got %d chunks, want at least 3", len(wire))
	}
	var reassembly reassembleChunks
	var message *completeMessage
	for i, frame := range chunkFrames(t, wire) {
		assembled, complete, err := reassembly.add(frame)
		if err != nil {
			t.Fatalf("a WriteRequest encoded into several chunks: feeding chunk %d failed: %v", i, err)
		}
		if complete {
			completed, ok := assembled.(*completeMessage)
			if !ok {
				t.Fatalf("a WriteRequest encoded into several chunks: reassembleChunks returned a %T, want a *completeMessage", assembled)
			}
			message = completed
		}
	}
	if message == nil {
		t.Fatalf("a WriteRequest encoded into several chunks: the final chunk returned no complete message, want the complete WriteRequest")
	}
	if message.requestID != requestID {
		t.Errorf("a WriteRequest encoded into several chunks: the reassembled message has request id %d, want %d", message.requestID, requestID)
	}
	write, ok := message.service.(*ua.WriteRequest)
	if !ok {
		t.Fatalf("a WriteRequest encoded into several chunks: the reassembled service is a %T, want a *ua.WriteRequest", message.service)
	}
	if len(write.NodesToWrite) != writeValueCount {
		t.Fatalf("a WriteRequest encoded into several chunks: the reassembled WriteRequest has %d NodesToWrite, want %d", len(write.NodesToWrite), writeValueCount)
	}
	for i, value := range write.NodesToWrite {
		if value.NodeID.String() != writeRequest.NodesToWrite[i].NodeID.String() {
			t.Errorf("a WriteRequest encoded into several chunks: NodesToWrite[%d].NodeID = %v, want the encoded node %v", i, value.NodeID, writeRequest.NodesToWrite[i].NodeID)
		}
		if value.AttributeID != ua.AttributeIDValue {
			t.Errorf("a WriteRequest encoded into several chunks: NodesToWrite[%d].AttributeID = %v, want %v", i, value.AttributeID, ua.AttributeIDValue)
		}
		valueString, ok := value.Value.Value.Value().(string)
		if !ok {
			t.Fatalf("a WriteRequest encoded into several chunks: NodesToWrite[%d] carries a %T value, want the encoded string", i, value.Value.Value.Value())
		}
		if valueString != fmt.Sprintf("spectest write value %d", i) {
			t.Errorf("a WriteRequest encoded into several chunks: NodesToWrite[%d] carries the value %q, want the encoded value", i, valueString)
		}
	}
	freshNode := ua.NewStringNodeID(1, "spectest.write.fresh")
	freshWire := encodeServiceChunks(t, requestID, math.MaxUint32, newReadRequest(requestID, freshNode))
	freshAssembled, freshComplete, err := reassembly.add(chunkFrames(t, freshWire)[0])
	if err != nil {
		t.Fatalf("a WriteRequest encoded into several chunks: reassembling a fresh message with the same request id failed, so the completed partial bodies were not dropped: %v", err)
	}
	if !freshComplete {
		t.Fatalf("a WriteRequest encoded into several chunks: the fresh message with the same request id returned no complete message, want the complete ReadRequest")
	}
	freshMessage, ok := freshAssembled.(*completeMessage)
	if !ok {
		t.Fatalf("a WriteRequest encoded into several chunks: the fresh message with the same request id reassembled to a %T, want a *completeMessage", freshAssembled)
	}
	freshRead, ok := freshMessage.service.(*ua.ReadRequest)
	if !ok {
		t.Fatalf("a WriteRequest encoded into several chunks: the fresh message with the same request id reassembled to a %T, want a *ua.ReadRequest", freshMessage.service)
	}
	if len(freshRead.NodesToRead) != 1 || freshRead.NodesToRead[0].NodeID.String() != freshNode.String() {
		t.Errorf("a WriteRequest encoded into several chunks: the fresh message with the same request id reads node %v, want the encoded node %v", freshRead.NodesToRead, freshNode)
	}
}

func TestReassembleChunksReassemblesInterleavedRequests(t *testing.T) {
	const firstRequestID = 10
	const secondRequestID = 20
	firstNode := ua.NewStringNodeID(1, "spectest.interleaved.first")
	secondNode := ua.NewStringNodeID(1, "spectest.interleaved.second")
	firstWire := encodeServiceChunks(t, firstRequestID, 32, newReadRequest(firstRequestID, firstNode))
	secondWire := encodeServiceChunks(t, secondRequestID, 32, newReadRequest(secondRequestID, secondNode))
	if len(firstWire) < 2 || len(secondWire) < 2 {
		t.Fatalf("two requests whose chunks interleave: got %d and %d chunks, want at least 2 each", len(firstWire), len(secondWire))
	}
	firstIntermediates := chunkFrames(t, firstWire[:len(firstWire)-1])
	secondIntermediates := chunkFrames(t, secondWire[:len(secondWire)-1])
	var reassembly reassembleChunks
	for i := 0; i < len(firstIntermediates) || i < len(secondIntermediates); i++ {
		for _, feed := range []struct {
			name   string
			frames []*chunkFrame
		}{{"first", firstIntermediates}, {"second", secondIntermediates}} {
			if i >= len(feed.frames) {
				continue
			}
			_, complete, err := reassembly.add(feed.frames[i])
			if err != nil {
				t.Fatalf("two requests whose chunks interleave: feeding intermediate chunk %d of the %s request failed: %v", i, feed.name, err)
			}
			if complete {
				t.Fatalf("two requests whose chunks interleave: feeding intermediate chunk %d of the %s request returned a complete message, want none", i, feed.name)
			}
		}
	}
	for _, request := range []struct {
		name       string
		requestID  uint32
		finalChunk *chunkFrame
		node       *ua.NodeID
	}{
		{"first", firstRequestID, chunkFrames(t, firstWire[len(firstWire)-1:])[0], firstNode},
		{"second", secondRequestID, chunkFrames(t, secondWire[len(secondWire)-1:])[0], secondNode},
	} {
		assembled, complete, err := reassembly.add(request.finalChunk)
		if err != nil {
			t.Fatalf("two requests whose chunks interleave: feeding the final chunk of the %s request failed: %v", request.name, err)
		}
		if !complete {
			t.Fatalf("two requests whose chunks interleave: feeding the final chunk of the %s request returned no complete message, want the complete ReadRequest", request.name)
		}
		message, ok := assembled.(*completeMessage)
		if !ok {
			t.Fatalf("two requests whose chunks interleave: the %s request reassembled to a %T, want a *completeMessage", request.name, assembled)
		}
		if message.requestID != request.requestID {
			t.Errorf("two requests whose chunks interleave: the reassembled %s request has request id %d, want %d", request.name, message.requestID, request.requestID)
		}
		read, ok := message.service.(*ua.ReadRequest)
		if !ok {
			t.Fatalf("two requests whose chunks interleave: the reassembled %s service is a %T, want a *ua.ReadRequest", request.name, message.service)
		}
		if len(read.NodesToRead) != 1 || read.NodesToRead[0].NodeID.String() != request.node.String() {
			t.Errorf("two requests whose chunks interleave: the reassembled %s request reads node %v, want the encoded node %v", request.name, read.NodesToRead, request.node)
		}
	}
}

func TestReassembleChunksReturnsAbortedOnAbortChunk(t *testing.T) {
	const abortedRequestID = 7
	wire := encodeServiceChunks(t, abortedRequestID, 32, newReadRequest(abortedRequestID, ua.NewStringNodeID(1, "spectest.aborted")))
	if len(wire) < 3 {
		t.Fatalf("a request aborted after intermediate chunks: got %d chunks, want at least 3 so the first ones are intermediate", len(wire))
	}
	var reassembly reassembleChunks
	for i, frame := range chunkFrames(t, wire[:len(wire)-1]) {
		_, complete, err := reassembly.add(frame)
		if err != nil {
			t.Fatalf("a request aborted after intermediate chunks: feeding intermediate chunk %d failed: %v", i, err)
		}
		if complete {
			t.Fatalf("a request aborted after intermediate chunks: feeding intermediate chunk %d returned a complete message, want none", i)
		}
	}
	abortBody, err := (&uasc.MessageAbort{ErrorCode: uint32(ua.StatusBadTimeout), Reason: "the request timed out"}).Encode()
	if err != nil {
		t.Fatalf("a request aborted after intermediate chunks: encoding the abort body failed: %v", err)
	}
	abortWire, err := messageChunk(uacp.ChunkTypeAbort, abortedRequestID, abortBody)
	if err != nil {
		t.Fatalf("a request aborted after intermediate chunks: encoding the abort chunk failed: %v", err)
	}
	abortedAssembled, aborted, err := reassembly.add(decodeChunk(t, 0, abortWire))
	if err != nil {
		t.Fatalf("a request aborted after intermediate chunks: feeding the abort chunk failed: %v", err)
	}
	if !aborted {
		t.Fatalf("a request aborted after intermediate chunks: feeding the abort chunk returned no aborted message, want one")
	}
	abortMessage, ok := abortedAssembled.(*abortedMessage)
	if !ok {
		t.Fatalf("a request aborted after intermediate chunks: feeding the abort chunk returned a %T, want an *abortedMessage", abortedAssembled)
	}
	if abortMessage.requestID != abortedRequestID {
		t.Errorf("a request aborted after intermediate chunks: the aborted message has request id %d, want %d", abortMessage.requestID, abortedRequestID)
	}
	if abortMessage.abort.ErrorCode != uint32(ua.StatusBadTimeout) {
		t.Errorf("a request aborted after intermediate chunks: the aborted message has status code %v, want %v", abortMessage.abort.ErrorCode, uint32(ua.StatusBadTimeout))
	}
	if abortMessage.abort.Reason != "the request timed out" {
		t.Errorf("a request aborted after intermediate chunks: the aborted message has reason %q, want %q", abortMessage.abort.Reason, "the request timed out")
	}
	freshNode := ua.NewStringNodeID(1, "spectest.aborted.fresh")
	freshWire := encodeServiceChunks(t, abortedRequestID, math.MaxUint32, newReadRequest(abortedRequestID, freshNode))
	freshAssembled, freshComplete, err := reassembly.add(chunkFrames(t, freshWire)[0])
	if err != nil {
		t.Fatalf("a request aborted after intermediate chunks: reassembling a fresh message with the same request id failed, so the aborted partial body was not dropped: %v", err)
	}
	if !freshComplete {
		t.Fatalf("a request aborted after intermediate chunks: the fresh message with the same request id returned no complete message, want the complete ReadRequest")
	}
	freshMessage, ok := freshAssembled.(*completeMessage)
	if !ok {
		t.Fatalf("a request aborted after intermediate chunks: the fresh message with the same request id reassembled to a %T, want a *completeMessage", freshAssembled)
	}
	freshRead, ok := freshMessage.service.(*ua.ReadRequest)
	if !ok {
		t.Fatalf("a request aborted after intermediate chunks: the fresh message with the same request id reassembled to a %T, want a *ua.ReadRequest", freshMessage.service)
	}
	if len(freshRead.NodesToRead) != 1 || freshRead.NodesToRead[0].NodeID.String() != freshNode.String() {
		t.Errorf("a request aborted after intermediate chunks: the fresh message with the same request id reads node %v, want the encoded node %v", freshRead.NodesToRead, freshNode)
	}
	const newRequestID = 42
	newNode := ua.NewStringNodeID(1, "spectest.aborted.new")
	newWire := encodeServiceChunks(t, newRequestID, math.MaxUint32, newReadRequest(newRequestID, newNode))
	newAssembled, newComplete, err := reassembly.add(chunkFrames(t, newWire)[0])
	if err != nil {
		t.Fatalf("a request aborted after intermediate chunks: reassembling a message with a new request id failed: %v", err)
	}
	if !newComplete {
		t.Fatalf("a request aborted after intermediate chunks: the message with the new request id returned no complete message, want the complete ReadRequest")
	}
	newMessage, ok := newAssembled.(*completeMessage)
	if !ok {
		t.Fatalf("a request aborted after intermediate chunks: the message with the new request id reassembled to a %T, want a *completeMessage", newAssembled)
	}
	newRead, ok := newMessage.service.(*ua.ReadRequest)
	if !ok {
		t.Fatalf("a request aborted after intermediate chunks: the message with the new request id reassembled to a %T, want a *ua.ReadRequest", newMessage.service)
	}
	if len(newRead.NodesToRead) != 1 || newRead.NodesToRead[0].NodeID.String() != newNode.String() {
		t.Errorf("a request aborted after intermediate chunks: the message with the new request id reads node %v, want the encoded node %v", newRead.NodesToRead, newNode)
	}
}

func TestReassembleChunksReturnsTruncatedRequestIDsAtClose(t *testing.T) {
	const firstRequestID = 0
	const secondRequestID = 7
	firstWire := encodeServiceChunks(t, firstRequestID, 32, newReadRequest(firstRequestID, ua.NewStringNodeID(1, "spectest.truncated.first")))
	if len(firstWire) < 2 {
		t.Fatalf("requests truncated at close: got %d chunks for the first request, want at least 2 so the first one is intermediate", len(firstWire))
	}
	secondWire := encodeServiceChunks(t, secondRequestID, 32, newReadRequest(secondRequestID, ua.NewStringNodeID(1, "spectest.truncated.second")))
	if len(secondWire) < 2 {
		t.Fatalf("requests truncated at close: got %d chunks for the second request, want at least 2 so the first one is intermediate", len(secondWire))
	}
	var reassembly reassembleChunks
	for _, feed := range []struct {
		name  string
		frame *chunkFrame
	}{
		{"the request with id 0", chunkFrames(t, firstWire[:1])[0]},
		{"the request with id 7", chunkFrames(t, secondWire[:1])[0]},
	} {
		_, complete, err := reassembly.add(feed.frame)
		if err != nil {
			t.Fatalf("requests truncated at close: feeding the intermediate chunk of %s failed: %v", feed.name, err)
		}
		if complete {
			t.Fatalf("requests truncated at close: feeding the intermediate chunk of %s returned a complete message, want none", feed.name)
		}
	}
	truncated := reassembly.truncatedAtClose()
	if len(truncated) != 2 || truncated[0] != firstRequestID || truncated[1] != secondRequestID {
		t.Fatalf("requests truncated at close: got %v, want exactly [%d %d], so request id 0 is reported", truncated, firstRequestID, secondRequestID)
	}
}

func TestTruncatedAtCloseWithNothingOutstanding(t *testing.T) {
	var clean reassembleChunks
	if truncated := clean.truncatedAtClose(); len(truncated) != 0 {
		t.Errorf("a close with nothing outstanding: got %v, want no truncated request ids", truncated)
	}
}

func TestReassembleChunksRejectsAnUndecodableCompleteMessage(t *testing.T) {
	const requestID = 42
	garbageWire, err := messageChunk(uacp.ChunkTypeFinal, requestID, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	if err != nil {
		t.Fatalf("a complete message whose body does not decode: encoding the garbage chunk failed: %v", err)
	}
	var reassembly reassembleChunks
	_, _, err = reassembly.add(decodeChunk(t, 0, garbageWire))
	if err == nil {
		t.Fatalf("a complete message whose body does not decode: reassembleChunks returned no error, want one naming request %d", requestID)
	}
	if !strings.Contains(err.Error(), "request 42") {
		t.Errorf("a complete message whose body does not decode: the error %q does not name request %d", err, requestID)
	}
}

func TestReassembleChunksRejectsAnUndecodableAbort(t *testing.T) {
	const requestID = 42
	abortWire, err := messageChunk(uacp.ChunkTypeAbort, requestID, []byte{0xff})
	if err != nil {
		t.Fatalf("an abort message whose body does not decode: encoding the abort chunk failed: %v", err)
	}
	var reassembly reassembleChunks
	_, _, err = reassembly.add(decodeChunk(t, 0, abortWire))
	if err == nil {
		t.Fatalf("an abort message whose body does not decode: reassembleChunks returned no error, want one naming request %d", requestID)
	}
	if !strings.Contains(err.Error(), "request 42") {
		t.Errorf("an abort message whose body does not decode: the error %q does not name request %d", err, requestID)
	}
}

func TestDecodeFrameCopiesTheChunkBodyOutOfTheStream(t *testing.T) {
	const requestID = 42
	wire := encodeServiceChunks(t, requestID, math.MaxUint32, newReadRequest(requestID, ua.NewStringNodeID(1, "spectest.copied")))[0]
	frame := decodeChunk(t, 0, wire)
	want := bytes.Clone(frame.body)
	for i := range wire {
		wire[i] = 0xff
	}
	if !bytes.Equal(frame.body, want) {
		t.Errorf("a chunk body that aliases the stream buffer: overwriting the stream buffer changed the chunk body, want decodeFrame to copy it")
	}
}

func TestDecodeFrameDecodesTransportMessages(t *testing.T) {
	errorBody, err := (&uacp.Error{ErrorCode: uint32(ua.StatusBadTimeout), Reason: "the connection timed out"}).Encode()
	if err != nil {
		t.Fatalf("an ERR message: encoding the error body failed: %v", err)
	}
	errorWire, err := transportMessage(uacp.MessageTypeError, errorBody)
	if err != nil {
		t.Fatalf("an ERR message: encoding the message failed: %v", err)
	}
	errorTransport := decodeTransport(t, errorWire)
	if errorTransport.messageType != uacp.MessageTypeError {
		t.Errorf("an ERR message: the transport frame has type %q, want %q", errorTransport.messageType, uacp.MessageTypeError)
	}
	errorDecoded, ok := errorTransport.body.(*uacp.Error)
	if !ok {
		t.Fatalf("an ERR message: the transport frame body is a %T, want the decoded *uacp.Error", errorTransport.body)
	}
	if errorDecoded.ErrorCode != uint32(ua.StatusBadTimeout) {
		t.Errorf("an ERR message: the transport frame has status code %v, want %v", errorDecoded.ErrorCode, uint32(ua.StatusBadTimeout))
	}
	if errorDecoded.Reason != "the connection timed out" {
		t.Errorf("an ERR message: the transport frame has reason %q, want %q", errorDecoded.Reason, "the connection timed out")
	}
	hello := &uacp.Hello{
		Version:        0,
		ReceiveBufSize: 65536,
		SendBufSize:    65536,
		MaxChunkCount:  1,
		EndpointURL:    "opc.tcp://127.0.0.1:4840",
	}
	helloBody, err := hello.Encode()
	if err != nil {
		t.Fatalf("a HEL message: encoding the hello body failed: %v", err)
	}
	helloWire, err := transportMessage(uacp.MessageTypeHello, helloBody)
	if err != nil {
		t.Fatalf("a HEL message: encoding the message failed: %v", err)
	}
	helloTransport := decodeTransport(t, helloWire)
	if helloTransport.messageType != uacp.MessageTypeHello {
		t.Errorf("a HEL message: the transport frame has type %q, want %q", helloTransport.messageType, uacp.MessageTypeHello)
	}
	helloDecoded, ok := helloTransport.body.(*uacp.Hello)
	if !ok {
		t.Fatalf("a HEL message: the transport frame body is a %T, want the decoded *uacp.Hello", helloTransport.body)
	}
	if helloDecoded.ReceiveBufSize != hello.ReceiveBufSize {
		t.Errorf("a HEL message: the decoded hello has ReceiveBufSize %d, want %d", helloDecoded.ReceiveBufSize, hello.ReceiveBufSize)
	}
	if helloDecoded.MaxChunkCount != hello.MaxChunkCount {
		t.Errorf("a HEL message: the decoded hello has MaxChunkCount %d, want %d", helloDecoded.MaxChunkCount, hello.MaxChunkCount)
	}
	if helloDecoded.EndpointURL != hello.EndpointURL {
		t.Errorf("a HEL message: the decoded hello has EndpointURL %q, want %q", helloDecoded.EndpointURL, hello.EndpointURL)
	}
	ack := &uacp.Acknowledge{
		Version:        0,
		ReceiveBufSize: 65536,
		SendBufSize:    65536,
		MaxMessageSize: 65536,
		MaxChunkCount:  1,
	}
	ackBody, err := ack.Encode()
	if err != nil {
		t.Fatalf("an ACK message: encoding the acknowledge body failed: %v", err)
	}
	ackWire, err := transportMessage(uacp.MessageTypeAcknowledge, ackBody)
	if err != nil {
		t.Fatalf("an ACK message: encoding the message failed: %v", err)
	}
	ackTransport := decodeTransport(t, ackWire)
	if ackTransport.messageType != uacp.MessageTypeAcknowledge {
		t.Errorf("an ACK message: the transport frame has type %q, want %q", ackTransport.messageType, uacp.MessageTypeAcknowledge)
	}
	ackDecoded, ok := ackTransport.body.(*uacp.Acknowledge)
	if !ok {
		t.Fatalf("an ACK message: the transport frame body is a %T, want the decoded *uacp.Acknowledge", ackTransport.body)
	}
	if ackDecoded.ReceiveBufSize != ack.ReceiveBufSize {
		t.Errorf("an ACK message: the decoded acknowledge has ReceiveBufSize %d, want %d", ackDecoded.ReceiveBufSize, ack.ReceiveBufSize)
	}
	if ackDecoded.MaxMessageSize != ack.MaxMessageSize {
		t.Errorf("an ACK message: the decoded acknowledge has MaxMessageSize %d, want %d", ackDecoded.MaxMessageSize, ack.MaxMessageSize)
	}
	if ackDecoded.MaxChunkCount != ack.MaxChunkCount {
		t.Errorf("an ACK message: the decoded acknowledge has MaxChunkCount %d, want %d", ackDecoded.MaxChunkCount, ack.MaxChunkCount)
	}
}

func decodeTransport(t *testing.T, wire []byte) *transportFrame {
	t.Helper()
	messages, rest, err := splitMessages(wire)
	if err != nil {
		t.Fatalf("decoding a transport message: splitMessages failed: %v", err)
	}
	if len(messages) != 1 || len(rest) != 0 {
		t.Fatalf("decoding a transport message: splitMessages returned %d messages and %d rest bytes, want 1 message and no rest", len(messages), len(rest))
	}
	decoded, err := decodeFrame(messages[0])
	if err != nil {
		t.Fatalf("decoding a transport message: decodeFrame failed: %v", err)
	}
	transport, ok := decoded.(*transportFrame)
	if !ok {
		t.Fatalf("decoding a transport message: decodeFrame returned a %T, want a *transportFrame", decoded)
	}
	return transport
}

func TestDecodeFrameRejectsMalformedMessages(t *testing.T) {
	chunkHeader := uasc.NewHeader(uasc.MessageTypeMessage, uacp.ChunkTypeFinal, 0)
	chunkHeader.MessageSize = 16
	chunkPrefix := ua.NewBuffer(nil)
	chunkPrefix.WriteStruct(chunkHeader)
	chunkPrefix.WriteStruct(uasc.NewSymmetricSecurityHeader(0))
	if chunkPrefix.Error() != nil {
		t.Fatalf("building malformed messages: encoding a chunk without a sequence header failed: %v", chunkPrefix.Error())
	}
	chunkWithoutSequenceHeader := chunkPrefix.Bytes()
	chunkTruncatedInsideItsHeaders := chunkWithoutSequenceHeader[:10]
	errorHeader := uacp.Header{MessageType: uacp.MessageTypeError, ChunkType: uacp.ChunkTypeFinal, MessageSize: headerLen + 4}
	errorHeaderBytes, err := errorHeader.Encode()
	if err != nil {
		t.Fatalf("building malformed messages: encoding the ERR header failed: %v", err)
	}
	errorWithoutReason := append(append([]byte{}, errorHeaderBytes...), 0xff, 0xff, 0xff, 0xff)
	helloBody, err := (&uacp.Hello{ReceiveBufSize: 65536, SendBufSize: 65536, MaxChunkCount: 1, EndpointURL: "opc.tcp://127.0.0.1:4840"}).Encode()
	if err != nil {
		t.Fatalf("building malformed messages: encoding the hello body failed: %v", err)
	}
	helloWire, err := transportMessage(uacp.MessageTypeHello, helloBody)
	if err != nil {
		t.Fatalf("building malformed messages: encoding the HEL message failed: %v", err)
	}
	helloTrailing := append(bytes.Clone(helloWire), 0x00)
	ackBody, err := (&uacp.Acknowledge{ReceiveBufSize: 65536, SendBufSize: 65536, MaxMessageSize: 65536, MaxChunkCount: 1}).Encode()
	if err != nil {
		t.Fatalf("building malformed messages: encoding the acknowledge body failed: %v", err)
	}
	ackWire, err := transportMessage(uacp.MessageTypeAcknowledge, ackBody)
	if err != nil {
		t.Fatalf("building malformed messages: encoding the ACK message failed: %v", err)
	}
	ackTrailing := append(bytes.Clone(ackWire), 0x00)
	errorBody, err := (&uacp.Error{ErrorCode: uint32(ua.StatusBadTimeout), Reason: "the connection timed out"}).Encode()
	if err != nil {
		t.Fatalf("building malformed messages: encoding the error body failed: %v", err)
	}
	errorWire, err := transportMessage(uacp.MessageTypeError, errorBody)
	if err != nil {
		t.Fatalf("building malformed messages: encoding the ERR message failed: %v", err)
	}
	errorTrailing := append(bytes.Clone(errorWire), 0x00)
	reverseHelloHeader := uacp.Header{MessageType: uacp.MessageTypeReverseHello, ChunkType: uacp.ChunkTypeFinal, MessageSize: headerLen}
	reverseHelloWire, err := reverseHelloHeader.Encode()
	if err != nil {
		t.Fatalf("building malformed messages: encoding the reverse hello header failed: %v", err)
	}
	cases := []struct {
		name      string
		message   []byte
		wantStage string
	}{
		{name: "a message shorter than the header", message: []byte("MSGF"), wantStage: "decoding message header"},
		{name: "a chunk truncated inside its headers", message: chunkTruncatedInsideItsHeaders, wantStage: "decoding chunk header"},
		{name: "a chunk without a sequence header", message: chunkWithoutSequenceHeader, wantStage: "decoding sequence header"},
		{name: "an ERR message without a decodable body", message: errorWithoutReason, wantStage: "decoding ERR message body"},
		{name: "a HEL message with one trailing byte", message: helloTrailing, wantStage: "decoding HEL message body: body is"},
		{name: "an ACK message with one trailing byte", message: ackTrailing, wantStage: "decoding ACK message body: body is"},
		{name: "an ERR message with one trailing byte", message: errorTrailing, wantStage: "decoding ERR message body: body is"},
		{name: "a reverse hello, which no frame represents", message: reverseHelloWire, wantStage: "cannot decode message type"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := decodeFrame(testCase.message)
			if err == nil {
				t.Fatalf("%s: decodeFrame returned no error, want one", testCase.name)
			}
			if !strings.Contains(err.Error(), testCase.wantStage) {
				t.Errorf("%s: decodeFrame error %q does not name the stage %q", testCase.name, err, testCase.wantStage)
			}
		})
	}
}
