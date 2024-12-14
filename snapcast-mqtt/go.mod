module github.com/claes/snapcast-mqtt

go 1.21.7

toolchain go1.22.8

require (
	github.com/ConnorsApps/snapcast-go v0.2.0
	github.com/eclipse/paho.mqtt.golang v1.5.0
)

require (
	github.com/claes/mqtt-bridges/common v0.0.0-00010101000000-000000000000 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/time v0.5.0 // indirect
)

replace github.com/claes/mqtt-bridges/common => ../common
