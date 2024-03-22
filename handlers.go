package framework

// handlers.go
// Everything required for commands to pass their own handlers to discordgo and the framework itself.

// dGOhandlers
// This list stores all the handlers that can be added to the bot
// It's basically a passthroughs for discordgo.AddHandler, but having a list
// allows them to be collected ahead of time and then added all at once
var dGOHandlers []interface{}

// AddDGOHandler
// This provides a way for commands to pass handler functions through to discordgo,
// and have them added properly during bot startup
func AddDGOHandler(handler interface{}) {
	dGOHandlers = append(dGOHandlers, handler)
}

// addHandlers
// Given all the handlers that have been pre-added to the handlers list, add them to the discordgo session
func addDGoHandlers() {
	if len(dGOHandlers) == 0 {
		return
	}

	for _, handler := range dGOHandlers {
		Session.AddHandler(handler)
	}
}
