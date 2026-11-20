module github.com/gopcua/opcua/contrib/changesink

go 1.24.0

require (
	github.com/eclipse/paho.mqtt.golang v1.5.1
	github.com/gopcua/opcua v0.8.0
	github.com/mochi-mqtt/server/v2 v2.7.9
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/xid v1.4.0 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gopcua/opcua => ../..
