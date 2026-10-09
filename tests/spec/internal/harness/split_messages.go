package harness

import (
	"fmt"
	"slices"

	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

const headerLen = 8

var acceptedMessageTypes = []string{
	uacp.MessageTypeHello,
	uacp.MessageTypeAcknowledge,
	uacp.MessageTypeError,
	uasc.MessageTypeMessage,
	uasc.MessageTypeOpenSecureChannel,
	uasc.MessageTypeCloseSecureChannel,
}

var acceptedChunkTypes = []byte{
	uacp.ChunkTypeIntermediate,
	uacp.ChunkTypeFinal,
	uacp.ChunkTypeAbort,
}

// The returned messages and rest alias stream, so copy them before
// reusing the buffer.
func splitMessages(stream []byte) (messages [][]byte, rest []byte, err error) {
	for offset := 0; offset < len(stream); {
		if len(stream)-offset < headerLen {
			return messages, stream[offset:], nil
		}
		var h uacp.Header
		_, _ = h.Decode(stream[offset : offset+headerLen])
		if !slices.Contains(acceptedMessageTypes, h.MessageType) {
			return messages, nil, fmt.Errorf("malformed header at offset %d: invalid message type %q", offset, h.MessageType)
		}
		if !slices.Contains(acceptedChunkTypes, h.ChunkType) {
			return messages, nil, fmt.Errorf("malformed header at offset %d: invalid chunk type %q", offset, h.ChunkType)
		}
		if h.MessageSize < headerLen {
			return messages, nil, fmt.Errorf("malformed header at offset %d: message size %d is less than the header length", offset, h.MessageSize)
		}
		if len(stream)-offset < int(h.MessageSize) {
			return messages, stream[offset:], nil
		}
		messages = append(messages, stream[offset:offset+int(h.MessageSize)])
		offset += int(h.MessageSize)
	}
	return messages, nil, nil
}
