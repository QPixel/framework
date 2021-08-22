package framework

import (
	"github.com/bwmarrin/discordgo"
	tlog "github.com/ubergeek77/tinylog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// core.go
// This file contains the main code responsible for driving core bot functionality

// MessageState
// Tells discordgo the amount of messages to cache
var MessageState = 500

// log
// The logger for the core bot
var log = tlog.NewTaggedLogger("BotCore", tlog.NewColor("38;5;111"))

// dlog
// The logger for discordgo
var dlog = tlog.NewTaggedLogger("DG", tlog.NewColor("38;5;111"))

// Session
// The Discord session, made public so commands can use it
var Session *discordgo.Session

// BotAdmins
// A list of user IDs that are designated as "Bot Administrators"
// These don't get saved to .json, and must be added programmatically
// They receive some privileges higher than guild moderators
// This is a boolean map, because checking its values is dead simple this way
var botAdmins = make(map[string]bool)

// BotToken
// A string of the current bot token, usually set by the main method
// Similar to BotAdmins, this isn't saved to .json and is added programmatically
var botToken = ""

// BotTestingId
// A string of the testing guild. Used for slash commands
var botTestingId = ""

// ColorSuccess
// The color to use for response embeds reporting success
var ColorSuccess = 0x55F485

// ColorFailure
// The color to use for response embeds reporting failure
var ColorFailure = 0xF45555

// BotPresence
// Presence data to send when the bot is logging in
var botPresence discordgo.GatewayStatusUpdate

// initProvider
// Stores and allows for the calling of the chosen GuildProvider
var initProvider func() GuildProvider

// SetInitProvider
// Sets the init provider
func SetInitProvider(provider func() GuildProvider) {
	initProvider = provider
	return
}

// SetPresence
// Sets the gateway field for bot presence
func SetPresence(presence discordgo.GatewayStatusUpdate) {
	botPresence = presence
	return
}

// AddAdmin
// A function that allows admins to be added, but not removed
func AddAdmin(userId string) {
	botAdmins[userId] = true
}

// SetToken
// A function that allows a single token to be added, but not removed
func SetToken(token string) {
	botToken = token
}

// SetTestingId
// A function that allows a single id to be added, but not removed
func SetTestingId(token string) {
	botTestingId = token
}

// IsAdmin
// Allow commands to check if a user is an admin or not
// Since botAdmins is a boolean map, if they are not in the map, false is the default
func IsAdmin(userId string) bool {
	return botAdmins[userId]
}

// IsCommand
// Check if a given string is a command registered to the core bot
func IsCommand(trigger string) bool {
	if _, ok := commands[strings.ToLower(trigger)]; ok {
		return true
	}
	return false
}

// Start the bot.
func Start() {
	discordgo.Logger = dgoLog

	// Load all the guilds
	if initProvider == nil {
		log.Fatalf("You have not chosen a database provider. Please refer to the docs")
	}
	currentProvider = initProvider()
	Guilds = loadGuilds()

	// We need a token
	if botToken == "" {
		log.Fatalf("You have not specified a Discord bot token!")
	}

	// Use the token to create a new session
	var err error
	Session, err = discordgo.New("Bot " + botToken)

	if err != nil {
		log.Fatalf("Failed to create Discord session: %s", err)
	}
	// Setup State specific variables
	Session.State.MaxMessageCount = MessageState
	Session.LogLevel = discordgo.LogWarning
	Session.SyncEvents = false
	Session.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAll)

	// Set the bots status
	Session.Identify.Presence = botPresence

	// Open the session
	log.Info("Connecting to Discord...")
	err = Session.Open()
	if err != nil {
		log.Fatalf("Failed to connect to Discord: %s", err)
	}

	// Add the commandHandler to the list of user-defined handlers
	AddDGOHandler(commandHandler)

	// Add the slash command handler to the list of user-defined handlers
	AddDGOHandler(handleInteraction)

	// Add the handlers to the session
	addDGoHandlers()

	// Log that the login succeeded
	log.Infof("Bot logged in as \"" + Session.State.Ready.User.Username + "#" + Session.State.Ready.User.Discriminator + "\"")

	// Start workers
	startWorkers()

	// Print information about the current bot admins
	numAdmins := 0
	for userId := range botAdmins {
		if user, err := GetUser(userId); err == nil {
			numAdmins += 1
			log.Infof("Added bot admin: %s#%s", user.Username, user.Discriminator)
		} else {
			log.Errorf("Unable to lookup bot admin user ID: " + userId)
		}
	}

	if numAdmins == 0 {
		log.Warning("You have not added any bot admins! Only moderators will be able to run commands, and permissions cannot be changed!")
	}

	//Register slash commands
	slashChannel := make(chan string)
	log.Info("Registering slash commands")
	go AddSlashCommands(botTestingId, slashChannel)

	// Bot ready
	log.Info("Initialization complete! The bot is now ready.")

	//Info about slash commands
	log.Info(<-slashChannel)

	// -- GRACEFUL TERMINATION -- //

	// Set up a sigterm channel, so we can detect when the application receives a TERM signal
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, os.Interrupt, os.Kill)

	// Keep this thread blocked forever, until a TERM signal is received
	<-sigChannel

	log.Info("Received TERM signal, terminating gracefully.")

	// Set the global loop variable to false so all background loops terminate
	continueLoop = false

	// Make a second sig channel that will respond to user term signal immediately
	sigInstant := make(chan os.Signal, 1)
	signal.Notify(sigInstant, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	// Make a goroutine that will wait for all background workers to be unlocked
	go func() {
		log.Info("Waiting for workers to exit... (interrupt to kill immediately; not recommended!!!)")
		for i, lock := range workerLock {
			// Try locking the worker mutex. This will block if the mutex is already locked
			// If we are able to lock it, then it means the worker has stopped.
			lock.Lock()
			log.Info("Stopped worker " + strconv.Itoa(i))
			lock.Unlock()
		}

		log.Info("All routines exited gracefully.")

		// Send our own signal to the instant sig channel
		sigInstant <- syscall.SIGTERM
	}()

	// Keep the thread blocked until the above goroutine finishes closing all workers, or until another TERM is received
	<-sigInstant

	log.Info("Closing the Discord session...")
	closeErr := Session.Close()
	if closeErr != nil {
		log.Errorf("An error occurred when closing the Discord session: %s", err)
		return
	}

	log.Info("Session closed.")
}
