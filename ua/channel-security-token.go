// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"time"
)

// ChannelSecurityToken represents a ChannelSecurityToken.
// It describes the new SecurityToken issued by the Server.
//
// Specification: Part 4, 5.5.2.2
// type ChannelSecurityToken struct {
// 	ChannelID       uint32
// 	TokenID         uint32
// 	CreatedAt       time.Time
// 	RevisedLifetime uint32
// }

// NewChannelSecurityToken creates a new ChannelSecurityToken.
func NewChannelSecurityToken(chanID, tokenID uint32, createdAt time.Time, lifetime uint32) *ChannelSecurityToken {
	return &ChannelSecurityToken{
		ChannelID:       chanID,
		TokenID:         tokenID,
		CreatedAt:       createdAt,
		RevisedLifetime: lifetime,
	}
}

// String returns ChannelSecurityToken in string.
func (c *ChannelSecurityToken) String() string {
	return fmt.Sprintf("%d, %d, %v, %d",
		c.ChannelID,
		c.TokenID,
		c.CreatedAt,
		c.RevisedLifetime,
	)
}
