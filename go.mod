module github.com/qpixel/framework

go 1.21

replace github.com/bwmarrin/discordgo => ../discordgo

require (
	github.com/QPixel/orderedmap v0.2.0
	github.com/bwmarrin/discordgo v0.27.2-0.20240315152229-33ee38cbf271
	github.com/dlclark/regexp2 v1.11.0
	github.com/ubergeek77/tinylog v1.0.0
	gitlab.com/tozd/go/errors v0.8.1
	golang.org/x/sys v0.18.0
)

require (
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.22.0 // indirect
)
