package framework

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/qpixel/framework/workers"
	tlog "github.com/ubergeek77/tinylog"
)

// core.go
// This file contains the main code responsible for driving core bot functionality

// MessageState
// Tells discordgo the amount of messages to cache
var MessageState = 500

// log
// The logger for the core bot
var log = tlog.NewTaggedLogger("Framework", tlog.NewColor("38;5;111"))

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

// initProvider
// Stores and allows for the calling of the chosen GuildProvider
var initProvider func() GuildProvider

// debugMode
// A boolean that tells the bot to log debug messages
var debugMode = false

// workerManager
// The worker manager for the bot
var WorkerManager *workers.WorkerManager

// SetInitProvider
// Sets the init provider
func SetInitProvider(provider func() GuildProvider) {
	initProvider = provider
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

// SetDebugMode
// Set the log level to debug
func SetDebugMode() {
	debugMode = true
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

	WorkerManager = workers.InitializeManager(time.UTC)

	// Use the token to create a new session
	var err error
	Session, err = discordgo.New("Bot " + botToken)

	if err != nil {
		log.Fatalf("Failed to create Discord session: %s", err)
	}
	if debugMode {
		Session.LogLevel = discordgo.LogInformational
		Session.Debug = true
	} else {
		Session.LogLevel = discordgo.LogWarning
	}

	if os.Getenv("LOG_LEVEL") != "" && os.Getenv("LOG_LEVEL") == "DEBUG" {
		Session.LogLevel = discordgo.LogDebug
	}

	// Setup State specific variables
	Session.State.MaxMessageCount = MessageState
	Session.SyncEvents = false
	Session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged | discordgo.IntentMessageContent

	// Add the commandHandler to the list of user-defined handlers
	AddDGOHandler(commandHandler)

	// Add the slash command handler to the list of user-defined handlers
	AddDGOHandler(handleInteraction)

	// Add the handlers to the session
	addDGoHandlers()

	// Open the session
	log.Info("Connecting to Discord...")
	err = Session.Open()
	if err != nil {
		log.Fatalf("Failed to connect to Discord: %s", err)
	}

	// Log that the login succeeded
	log.Infof("Bot logged in as \"" + Session.State.Ready.User.Username + "#" + Session.State.Ready.User.Discriminator + "\"")

	// Start workers

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
	go RegisterSlashCommands(botTestingId, slashChannel)

	// Bot ready
	log.Info("Initialization complete! The bot is now ready.")

	//Info about slash commands
	log.Info(<-slashChannel)

	// -- GRACEFUL TERMINATION -- //

	// Set up a sigterm channel, so we can detect when the application receives a TERM signal
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Keep this thread blocked forever, until a TERM signal is received
	<-sigChannel

	log.Info("Received TERM signal, terminating gracefully.")

	// Make a second sig channel that will respond to user term signal immediately
	sigInstant := make(chan os.Signal, 1)
	signal.Notify(sigInstant, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Make a goroutine that will wait for all background workers to be unlocked
	go func() {
		log.Info("Waiting for workers to exit... (interrupt to kill immediately; not recommended!!!)")
		// Stop all workers
		WorkerManager.StopWorkers()
		log.Info("All routines exited gracefully.")

		// Send our own signal to the instant sig channel
		sigInstant <- syscall.SIGTERM
	}()

	// Keep the thread blocked until the above goroutine finishes closing all workers, or until another TERM is received
	<-sigInstant

	log.Info("Closing the Discord session...")
	closeErr := Session.CloseWithCode(1000)
	if closeErr != nil {
		log.Errorf("An error occurred when closing the Discord session: %s", err)
		return
	}

	log.Info("Session closed.")
}
