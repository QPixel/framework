package framework

// handlers.go
// Everything required for commands to pass their own handlers to discordgo

// handlers
// This list stores all the handlers that can be added to the bot
// It's basically a passthroughs for discordgo.AddHandler, but having a list
// allows them to be collected ahead of time and then added all at once
var handlers []interface{}

// AddHandler
// This provides a way for commands to pass handler functions through to discorgo,
// and have them added properly during bot startup
func AddHandler(handler interface{}) {
	handlers = append(handlers, handler)
}

// addHandlers
// Given all the handlers that have been pre-added to the handlers list, add them to the discordgo session
func addHandlers() {
	if len(handlers) == 0 {
		return
	}

	for _, handler := range handlers {
		Session.AddHandler(handler)
	}
}
