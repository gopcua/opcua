package uasc

import (
	"math"
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uapolicy"
	"github.com/stretchr/testify/require"
)

func TestSetMaximumBodySize_Padding(t *testing.T) {
	tests := []struct {
		name           string
		securityPolicy string
		nonceLength    int
		messageSize    int
		expectedBody   uint32
	}{
		{
			name:           "No Security",
			securityPolicy: ua.SecurityPolicyURINone,
			nonceLength:    0,
			messageSize:    8192,
			expectedBody: maxBodySizeAccordingToSpec(maxBodySizeArgs{
				plaintextBlockSize:  1,
				messageChunkSize:    8192,
				headerSize:          12,
				cipherTextBlockSize: 1,
				sequenceHeaderSize:  8,
				signatureSize:       0,
			}),
		},
		{
			name:           "Basic128Rsa15",
			securityPolicy: ua.SecurityPolicyURIBasic128Rsa15,
			nonceLength:    16,
			messageSize:    8192,
			expectedBody: maxBodySizeAccordingToSpec(maxBodySizeArgs{
				plaintextBlockSize:  16,
				messageChunkSize:    8192,
				headerSize:          12,
				cipherTextBlockSize: 16,
				sequenceHeaderSize:  8,
				signatureSize:       160 / 8,
			}),
		},
		{
			name:           "Basic256",
			securityPolicy: ua.SecurityPolicyURIBasic256,
			nonceLength:    32,
			messageSize:    8192,
			expectedBody: maxBodySizeAccordingToSpec(maxBodySizeArgs{
				plaintextBlockSize:  16,
				messageChunkSize:    8192,
				headerSize:          12,
				cipherTextBlockSize: 16,
				sequenceHeaderSize:  8,
				signatureSize:       160 / 8,
			}),
		},
		{
			name:           "Basic256Sha256",
			securityPolicy: ua.SecurityPolicyURIBasic256Sha256,
			nonceLength:    32,
			messageSize:    8192,
			expectedBody: maxBodySizeAccordingToSpec(maxBodySizeArgs{
				plaintextBlockSize:  16,
				messageChunkSize:    8192,
				headerSize:          12,
				cipherTextBlockSize: 16,
				sequenceHeaderSize:  8,
				signatureSize:       256 / 8,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeNonce := make([]byte, tt.nonceLength)
			algo, err := uapolicy.Symmetric(tt.securityPolicy, fakeNonce, fakeNonce)
			require.NoError(t, err)

			ch := &channelInstance{algo: algo}
			ch.setMaximumBodySize(tt.messageSize)
			require.NoError(t, err)
			require.Equal(t, int(tt.expectedBody), int(ch.maxBodySize))
		})
	}
}

type maxBodySizeArgs struct {
	plaintextBlockSize  int
	messageChunkSize    int
	headerSize          int
	cipherTextBlockSize int
	sequenceHeaderSize  int
	signatureSize       int
}

// https://reference.opcfoundation.org/v104/Core/docs/Part6/6.7.2/
// The formula to calculate the amount of padding depends on the amount of data that needs to be sent (called BytesToWrite). The sender shall first calculate the maximum amount of space available in the MessageChunk (called MaxBodySize) using the following formula:
//
// MaxBodySize = PlainTextBlockSize * Floor ((MessageChunkSize – HeaderSize - 1)/CipherTextBlockSize) –
//
//     SequenceHeaderSize – SignatureSize;

func maxBodySizeAccordingToSpec(args maxBodySizeArgs) uint32 {
	return uint32(args.plaintextBlockSize*int(math.Floor(float64(args.messageChunkSize-args.headerSize-1)/float64(args.cipherTextBlockSize))) - args.sequenceHeaderSize - args.signatureSize)
}
