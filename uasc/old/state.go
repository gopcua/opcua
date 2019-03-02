// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

// type secChanState uint8

// const (
// 	undefined secChanState = iota
// 	transportUnavailable
// 	cliStateSecureChannelClosed
// 	cliStateOpenSecureChannelSent
// 	cliStateSecureChannelOpened
// 	cliStateCloseSecureChannelSent
// 	srvStateSecureChannelClosed
// 	srvStateSecureChannelOpened
// 	srvStateCloseSecureChannelSent
// )

// func (s secChanState) String() string {
// 	switch s {
// 	case transportUnavailable:
// 		return "transport connection unavailable"
// 	case cliStateSecureChannelClosed:
// 		return "client secure channel closed"
// 	case cliStateOpenSecureChannelSent:
// 		return "client open secure channel sent"
// 	case cliStateSecureChannelOpened:
// 		return "client secure channel opened"
// 	case cliStateCloseSecureChannelSent:
// 		return "client close secure channel sent"
// 	case srvStateSecureChannelClosed:
// 		return "server secure channel closed"
// 	case srvStateSecureChannelOpened:
// 		return "server secure channel opened"
// 	case srvStateCloseSecureChannelSent:
// 		return "server close secure channel sent"
// 	default:
// 		return "unknown"
// 	}
// }

// // GetState returns the current secChanState of SecureChannel.
// func (s *SecureChannel) GetState() string {
// 	if s == nil {
// 		return ""
// 	}
// 	return s.state.String()
// }

// type sessionState uint8

// const (
// 	cliStateSessionClosed sessionState = iota
// 	cliStateCreateSessionSent
// 	cliStateSessionCreated
// 	cliStateActivateSessionSent
// 	cliStateSessionActivated
// 	cliStateCloseSessionSent
// 	srvStateSessionClosed
// 	srvStateSessionCreated
// 	srvStateActivateSessionSent
// 	srvStateSessionActivated
// 	srvStateCloseSessionSent
// )

// func (s sessionState) String() string {
// 	switch s {
// 	case cliStateSessionClosed:
// 		return "cliStateSessionClosed"
// 	case cliStateCreateSessionSent:
// 		return "cliStateCreateSessionSent"
// 	case cliStateSessionCreated:
// 		return "cliStateSessionCreated"
// 	case cliStateActivateSessionSent:
// 		return "cliStateActivateSessionSent"
// 	case cliStateSessionActivated:
// 		return "cliStateSessionActivated"
// 	case cliStateCloseSessionSent:
// 		return "cliStateCloseSessionSent"
// 	case srvStateSessionClosed:
// 		return "srvStateSessionClosed"
// 	case srvStateSessionCreated:
// 		return "srvStateSessionCreated"
// 	case srvStateActivateSessionSent:
// 		return "srvStateActivateSessionSent"
// 	case srvStateSessionActivated:
// 		return "srvStateSessionActivated"
// 	case srvStateCloseSessionSent:
// 		return "srvStateCloseSessionSent"
// 	default:
// 		return ""
// 	}
// }
