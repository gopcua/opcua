package harness

import (
	"fmt"
	"maps"
	"slices"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

type frame interface {
	isFrame()
}

type transportFrame struct {
	messageType string
	body        any
}

func (*transportFrame) isFrame() {}

type chunkFrame struct {
	chunkType byte
	requestID uint32
	body      []byte
}

func (*chunkFrame) isFrame() {}

type assembled interface {
	isAssembled()
}

type completeMessage struct {
	requestID uint32
	service   any
}

func (*completeMessage) isAssembled() {}

type abortedMessage struct {
	requestID uint32
	abort     *uasc.MessageAbort
}

func (*abortedMessage) isAssembled() {}

type reassembleChunks struct {
	partial map[uint32][][]byte
}

func decodeFrame(message []byte) (frame, error) {
	if len(message) < headerLen {
		return nil, fmt.Errorf("decoding message header: message is %d bytes, want at least %d", len(message), headerLen)
	}
	var header uacp.Header
	if _, err := header.Decode(message[:headerLen]); err != nil {
		return nil, fmt.Errorf("decoding message header: %w", err)
	}
	switch header.MessageType {
	case uasc.MessageTypeOpenSecureChannel, uasc.MessageTypeMessage, uasc.MessageTypeCloseSecureChannel:
		chunk := &uasc.MessageChunk{}
		if _, err := chunk.Decode(message); err != nil {
			return nil, fmt.Errorf("decoding chunk header: %w", err)
		}
		sequenceHeader := &uasc.SequenceHeader{}
		sequenceHeaderLen, err := sequenceHeader.Decode(chunk.Data)
		if err != nil {
			return nil, fmt.Errorf("decoding sequence header: %w", err)
		}
		return &chunkFrame{
			chunkType: chunk.ChunkType,
			requestID: sequenceHeader.RequestID,
			body:      slices.Clone(chunk.Data[sequenceHeaderLen:]),
		}, nil
	case uacp.MessageTypeHello:
		return decodeTransportBody(header.MessageType, &uacp.Hello{}, message)
	case uacp.MessageTypeAcknowledge:
		return decodeTransportBody(header.MessageType, &uacp.Acknowledge{}, message)
	case uacp.MessageTypeError:
		return decodeTransportBody(header.MessageType, &uacp.Error{}, message)
	}
	return nil, fmt.Errorf("cannot decode message type %q", header.MessageType)
}

func decodeTransportBody(messageType string, body interface{ Decode([]byte) (int, error) }, message []byte) (*transportFrame, error) {
	decodedLen, err := body.Decode(message[headerLen:])
	if err != nil {
		return nil, fmt.Errorf("decoding %s message body: %w", messageType, err)
	}
	if decodedLen != len(message)-headerLen {
		return nil, fmt.Errorf("decoding %s message body: body is %d bytes, decoder consumed %d", messageType, len(message)-headerLen, decodedLen)
	}
	return &transportFrame{messageType: messageType, body: body}, nil
}

func (r *reassembleChunks) add(frame *chunkFrame) (assembled, bool, error) {
	switch frame.chunkType {
	case uacp.ChunkTypeIntermediate:
		if r.partial == nil {
			r.partial = make(map[uint32][][]byte)
		}
		r.partial[frame.requestID] = append(r.partial[frame.requestID], frame.body)
		return nil, false, nil
	case uacp.ChunkTypeAbort:
		delete(r.partial, frame.requestID)
		abort := &uasc.MessageAbort{}
		if _, err := abort.Decode(frame.body); err != nil {
			return nil, false, fmt.Errorf("decoding abort of request %d: %w", frame.requestID, err)
		}
		return &abortedMessage{requestID: frame.requestID, abort: abort}, true, nil
	}
	bodies := r.partial[frame.requestID]
	delete(r.partial, frame.requestID)
	var body []byte
	for _, partialBody := range bodies {
		body = append(body, partialBody...)
	}
	body = append(body, frame.body...)
	_, service, err := ua.DecodeService(body)
	if err != nil {
		return nil, false, fmt.Errorf("decoding body of request %d: %w", frame.requestID, err)
	}
	return &completeMessage{requestID: frame.requestID, service: service}, true, nil
}

func (r *reassembleChunks) truncatedAtClose() []uint32 {
	return slices.Sorted(maps.Keys(r.partial))
}
