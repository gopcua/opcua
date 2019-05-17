// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	nodeID := flag.String("node", "", "NodeID to read")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	c := opcua.NewClient(*endpoint)
	if err := c.Connect(); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	node, err := ua.ParseNodeID(*nodeID)
	if err != nil {
		log.Fatalf("invalid node id: %v", err)
	}

	// HistoryRead with ContinuationPoint use
	nodesToRequest := make([]*ua.HistoryReadValueID, 1)
	nodesToRequest[0] = &ua.HistoryReadValueID{
		NodeID:       node,
		DataEncoding: &ua.QualifiedName{},
	}

	for {
		if len(nodesToRequest) < 1 {
			break
		}

		// For ContinuationPoint usage
		// Get current nodes
		nodes := nodesToRequest
		// Reset old nodes
		nodesToRequest = make([]*ua.HistoryReadValueID, 0)

		// Part 4, 5.10.3 HistoryRead
		req := &ua.HistoryReadRequest{
			TimestampsToReturn:        ua.TimestampsToReturnSource,
			ReleaseContinuationPoints: false,
			NodesToRead:               nodes,
			// Part 11, 6.4 HistoryReadDetails parameters
			HistoryReadDetails: &ua.ExtensionObject{
				TypeID:       ua.NewFourByteExpandedNodeID(0, id.ReadRawModifiedDetails_Encoding_DefaultBinary),
				EncodingMask: ua.ExtensionObjectBinary,
				Value: &ua.ReadRawModifiedDetails{
					IsReadModified:   false,
					StartTime:        time.Now().UTC().AddDate(0, -1, 0),
					EndTime:          time.Now().UTC().AddDate(0, 1, 0),
					NumValuesPerNode: 0,
					ReturnBounds:     false,
				},
			},
		}

		data := &ua.HistoryReadResponse{}
		err := c.Send(req, func(v interface{}) error {
			ok := false
			if data, ok = v.(*ua.HistoryReadResponse); ok {
				return nil
			}

			return fmt.Errorf("cant parse response")
		})

		if err != nil {
			log.Printf("HistoryReadRequest error: %s", err)
			break
		}

		if data == nil || len(data.Results) < 1 {
			break
		}

		for nodeNum, result := range data.Results {
			if result.StatusCode != ua.StatusOK {
				log.Printf("result.StatusCode not StatusOK: %d", result.StatusCode)
				continue
			}

			// If this is not the end of the data - continue requesting
			if len(result.ContinuationPoint) > 0 {
				nodeNew := nodes[nodeNum]
				nodeNew.ContinuationPoint = result.ContinuationPoint
				nodesToRequest = append(nodesToRequest, nodeNew)
			}

			if result.HistoryData == nil || result.HistoryData.Value == nil {
				continue
			}

			historyData, ok := result.HistoryData.Value.(*ua.HistoryData)
			if !ok || historyData == nil || len(historyData.DataValues) < 1 {
				continue
			}

			for _, value := range historyData.DataValues {
				log.Printf(
					"%s - %s - %v \n",
					nodes[nodeNum].NodeID.String(),
					value.SourceTimestamp.Format(time.RFC3339),
					value.Value.Value,
				)
			}
		}
	}
}
