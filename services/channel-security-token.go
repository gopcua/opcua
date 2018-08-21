// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/errors"
)

// ChannelSecurityToken represents a ChannelSecurityToken.
type ChannelSecurityToken struct {
	ChannelID       uint32
	TokenID         uint32
	CreatedAt       uint64
	RevisedLifetime uint32
}

// NewChannelSecurityToken creates a new ChannelSecurityToken.
func NewChannelSecurityToken(channel, token uint32, createdAt uint64, lifetime uint32) *ChannelSecurityToken {
	return &ChannelSecurityToken{
		ChannelID:       channel,
		TokenID:         token,
		CreatedAt:       createdAt,
		RevisedLifetime: lifetime,
	}
}

// DecodeChannelSecurityToken decodes given bytes into ChannelSecurityToken.
func DecodeChannelSecurityToken(b []byte) (*ChannelSecurityToken, error) {
	c := &ChannelSecurityToken{}
	if err := c.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return c, nil
}

// DecodeFromBytes decodes given bytes into ChannelSecurityToken.
func (c *ChannelSecurityToken) DecodeFromBytes(b []byte) error {
	if len(b) < 20 {
		return &errors.ErrTooShortToDecode{c, "should be longer than 20 bytes"}
	}

	c.ChannelID = binary.LittleEndian.Uint32(b[0:4])
	c.TokenID = binary.LittleEndian.Uint32(b[4:8])
	c.CreatedAt = binary.LittleEndian.Uint64(b[8:16])
	c.RevisedLifetime = binary.LittleEndian.Uint32(b[16:20])

	return nil
}

// Serialize serializes ChannelSecurityToken into bytes.
func (c *ChannelSecurityToken) Serialize() ([]byte, error) {
	b := make([]byte, c.Len())
	if err := c.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ChannelSecurityToken into bytes.
func (c *ChannelSecurityToken) SerializeTo(b []byte) error {
	binary.LittleEndian.PutUint32(b[:4], c.ChannelID)
	binary.LittleEndian.PutUint32(b[4:8], c.TokenID)
	binary.LittleEndian.PutUint64(b[8:16], c.CreatedAt)
	binary.LittleEndian.PutUint32(b[16:20], c.RevisedLifetime)

	return nil
}

// Len returns the actual length of ChannelSecurityToken.
func (c *ChannelSecurityToken) Len() int {
	return 20
}

// String returns ChannelSecurityToken in string.
func (c *ChannelSecurityToken) String() string {
	return fmt.Sprintf("%d, %d, %d, %d",
		c.ChannelID,
		c.TokenID,
		c.CreatedAt,
		c.RevisedLifetime,
	)
}
