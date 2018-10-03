// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
)

// ServersOnNetwork is a DNS service record that meet criteria specified in the request.
// This list is empty if no Servers meet the criteria.
//
// Specification: Part4, 5.4.3.2
type ServersOnNetwork struct {
	RecordID           uint32
	ServerName         *String
	DiscoveryURI       *String
	ServerCapabilities *StringArray
}

// NewServersOnNetwork creates a new NewServersOnNetwork.
func NewServersOnNetwork(record uint32, serverName, discoveryURI string, serverCap []string) *ServersOnNetwork {
	return &ServersOnNetwork{
		RecordID:           record,
		ServerName:         NewString(serverName),
		DiscoveryURI:       NewString(discoveryURI),
		ServerCapabilities: NewStringArray(serverCap),
	}
}

// DecodeServersOnNetwork decodes given bytes into ServersOnNetwork.
func DecodeServersOnNetwork(b []byte) (*ServersOnNetwork, error) {
	s := &ServersOnNetwork{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into OPC UA ServersOnNetwork.
func (s *ServersOnNetwork) DecodeFromBytes(b []byte) error {
	offset := 0
	s.RecordID = uint32(binary.LittleEndian.Uint32(b[offset:]))
	offset += 4

	s.ServerName = &String{}
	if err := s.ServerName.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += s.ServerName.Len()

	s.DiscoveryURI = &String{}
	if err := s.DiscoveryURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += s.DiscoveryURI.Len()

	s.ServerCapabilities = &StringArray{}
	if err := s.ServerCapabilities.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}

	return nil
}

// Serialize serializes ServersOnNetwork into bytes.
func (s *ServersOnNetwork) Serialize() ([]byte, error) {
	b := make([]byte, s.Len())
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ServersOnNetwork into bytes.
func (s *ServersOnNetwork) SerializeTo(b []byte) error {
	offset := 0

	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(s.RecordID))
	offset += 4

	if s.ServerName != nil {
		if err := s.ServerName.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += s.ServerName.Len()
	}

	if s.DiscoveryURI != nil {
		if err := s.DiscoveryURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += s.DiscoveryURI.Len()
	}

	if s.ServerCapabilities != nil {
		return s.ServerCapabilities.SerializeTo(b[offset:])
	}

	return nil
}

// Len returns the actual length of ServersOnNetwork in int.
func (s *ServersOnNetwork) Len() int {
	l := 4

	if s.ServerName != nil {
		l += s.ServerName.Len()
	}

	if s.DiscoveryURI != nil {
		l += s.DiscoveryURI.Len()
	}

	if s.ServerCapabilities != nil {
		l += s.ServerCapabilities.Len()
	}

	return l
}

// ServersOnNetworkArray represents an array of ServersOnNetworks.
// It does not correspond to a certain type from the specification
// but makes encoding and decoding easier.
type ServersOnNetworkArray struct {
	ArraySize         int32
	ServersOnNetworks []*ServersOnNetwork
}

// NewServersOnNetworkArray creates a new ServersOnNetworkArray from multiple ServersOnNetworks.
func NewServersOnNetworkArray(ids []*ServersOnNetwork) *ServersOnNetworkArray {
	if ids == nil {
		r := &ServersOnNetworkArray{
			ArraySize: 0,
		}
		return r
	}

	r := &ServersOnNetworkArray{
		ArraySize:         int32(len(ids)),
		ServersOnNetworks: ids,
	}

	return r
}

// DecodeServersOnNetworkArray decodes given bytes into ServersOnNetworkArray.
func DecodeServersOnNetworkArray(b []byte) (*ServersOnNetworkArray, error) {
	s := &ServersOnNetworkArray{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return s, nil
}

// DecodeFromBytes decodes given bytes into ServersOnNetworkArray.
func (s *ServersOnNetworkArray) DecodeFromBytes(b []byte) error {
	s.ArraySize = int32(binary.LittleEndian.Uint32(b[:4]))
	if s.ArraySize <= 0 {
		return nil
	}

	offset := 4
	for i := 1; i <= int(s.ArraySize); i++ {
		id, err := DecodeServersOnNetwork(b[offset:])
		if err != nil {
			return err
		}
		s.ServersOnNetworks = append(s.ServersOnNetworks, id)
		offset += id.Len()
	}

	return nil
}

// Serialize serializes ServersOnNetworkArray into bytes.
func (s *ServersOnNetworkArray) Serialize() ([]byte, error) {
	b := make([]byte, s.Len())
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ServersOnNetworkArray into bytes.
func (s *ServersOnNetworkArray) SerializeTo(b []byte) error {
	var offset = 4
	binary.LittleEndian.PutUint32(b[:4], uint32(s.ArraySize))

	for _, id := range s.ServersOnNetworks {
		if err := id.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += id.Len()
	}

	return nil
}

// Len returns the actual length in int.
func (s *ServersOnNetworkArray) Len() int {
	l := 4
	for _, id := range s.ServersOnNetworks {
		l += id.Len()
	}
	return l
}
