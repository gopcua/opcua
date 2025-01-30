package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var port = 4840

	// Server options.
	opts := []server.Option{
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),

		// Endpoints.
		server.EndPoint("192.168.2.66", port),
		server.EndPoint("opcua", port),
	}

	// Create the server.
	s := server.New(opts...)

	// Add the root namespace.
	rootNS, _ := s.Namespace(0)
	rootObjNS := rootNS.Objects()

	// Add the tag namespace.
	tagNS := server.NewNodeNameSpace(s, "Tags")
	tagObjNS := tagNS.Objects()
	rootObjNS.AddRef(tagObjNS, id.HasComponent, true)

	// Add a tag.
	val := ua.DataValue{
		Value:           ua.MustVariant(206.1),
		Status:          ua.StatusBad,
		SourceTimestamp: time.Now(),
		EncodingMask:    ua.DataValueValue | ua.DataValueStatusCode | ua.DataValueSourceTimestamp,
	}
	n := tagNS.AddNewVariableNode("Voltage", val)
	tagObjNS.AddRef(n, id.HasComponent, true)

	if err := s.Start(context.Background()); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}

	// simulate a background process updating the data in the namespace.
	go func() {
		time.Sleep(time.Second * 10)
		for {
			// wrap the new value in a DataValue and use that to update the Value attribute of the node
			val := ua.DataValue{
				Value:           ua.MustVariant(206.1),
				Status:          ua.StatusBad,
				SourceTimestamp: time.Now(),
				EncodingMask:    ua.DataValueValue | ua.DataValueStatusCode | ua.DataValueSourceTimestamp,
			}
			n.SetAttribute(ua.AttributeIDValue, &val)

			// we also need to let the node namespace know that the value has changed so it can trigger the change notification
			// and send the updated value to any subscribed clients.
			tagNS.ChangeNotification(n.ID())

			time.Sleep(time.Second)
		}
	}()

	// catch ctrl-c and gracefully shutdown the server.
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	defer signal.Stop(sigch)
	log.Printf("Press CTRL-C to exit")

	<-sigch
	log.Printf("Shutting down the server...")

}
