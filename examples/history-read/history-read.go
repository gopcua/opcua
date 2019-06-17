// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"
)

func main() {
	endpoint := flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	nodeID := flag.String("node", "", "NodeID to read")
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	ctx := context.Background()

	c := opcua.NewClient(*endpoint)
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	id, err := ua.ParseNodeID(*nodeID)
	if err != nil {
		log.Fatalf("invalid node id: %v", err)
	}

	// HistoryRead with ContinuationPoint use
	nodesToRequest := []*ua.HistoryReadValueID{
		&ua.HistoryReadValueID{
			NodeID:       id,
			DataEncoding: &ua.QualifiedName{},
		},
	}

	for {
		if len(nodesToRequest) == 0 {
			break
		}

		// For ContinuationPoint usage
		// Get current nodes
		nodes := nodesToRequest
		// Reset old nodes
		nodesToRequest = make([]*ua.HistoryReadValueID, 0)

		data, err := c.HistoryReadRawModified(nodes, &ua.ReadRawModifiedDetails{
			IsReadModified: false,
			StartTime:      time.Now().UTC().AddDate(0, -1, 0),
			EndTime:        time.Now().UTC().AddDate(0, 1, 0),
		})
		if err != nil {
			log.Printf("HistoryReadRequest error: %s", err)
			break
		}

		if data == nil || len(data.Results) == 0 {
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
			if !ok || historyData == nil || len(historyData.DataValues) == 0 {
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
