package framework

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/QPixel/orderedmap"
	"github.com/bwmarrin/discordgo"
	"github.com/dlclark/regexp2"
)

// commands.go
// This file contains everything required to add core commands to the bot, and parse commands from a message

// Group
// Defines different "groups" of commands for ordering in a help command
type Group string

var (
	Moderation     Group = "moderation"
	Utility        Group = "utility"
	UserContext    Group = "context"
	MessageContext Group = "message"
)

type CommandType string

var (
	ChatCommand    CommandType = "CHAT"
	UserCommand    CommandType = "USER"
	MessageCommand CommandType = "MESSAGE"
)

// CommandInfo
// The definition of a command's info. This is everything about the command, besides the function it will run
type CommandInfo struct {
	Type                 CommandType            // The type of command
	Aliases              []string               // Aliases for the normal trigger
	Arguments            *orderedmap.OrderedMap // Arguments for the command
	Description          string                 // A short description of what the command does
	Group                Group                  // The group this command belongs to
	ParentID             string                 // The ID of the parent command
	Public               bool                   // Whether non-admins and non-mods can use this command
	IsTyping             bool                   // Whether the command will show a typing thing when ran.
	IsParent             bool                   // If the command is the parent of a subcommand tree
	IsChild              bool                   // If the command is the child
	Name                 string                 // The name of the command
	IntegrationTypes     []discordgo.ApplicationIntegrationType
	InstallationContexts []discordgo.InteractionContextType
}

// Context
// This is a context of a single command invocation
// This gives the command function access to all the information it might need
type Context struct {
	Guild       *Guild // NOTE: Guild is a pointer, since we want to use the SAME instance of the guild across the program!
	Cmd         CommandInfo
	Args        Arguments
	Message     *discordgo.Message
	Interaction *discordgo.Interaction
}

// BotFunction
// This type defines the functions that are called when commands are triggered
// Contexts are also passed as pointers, so they are not re-allocated when passed through
type BotFunction func(ctx *Context)

// Command
// The definition of a command, which is that command's information, along with the functions it will run
// Handlers is a map of strings to BotFunctions, so that different handlers can be used for different situations
type Command struct {
	Info               *CommandInfo
	Handlers           map[string]BotFunction
	ApplicationCommand *discordgo.ApplicationCommand
}

// commands
// All commands that are registered with the bot are stored here
// This is private so that other commands cannot modify it
var commands = make(map[string]*Command)

// Command Aliases
// A map of aliases to command triggers
var commandAliases = make(map[string]string)

// component handlers
var componentHandlers = make(map[string]BotFunction)

// commandsGC
var commandsGC = 0

// -- Command Configuration --

// CreateCommandInfo
// Creates a pointer to a CommandInfo
func CreateCommandInfo(name string, description string, public bool, group Group, command_type ...CommandType) *CommandInfo {
	if len(command_type) < 1 {
		command_type = append(command_type, ChatCommand)
	}
	cI := &CommandInfo{
		Type:        command_type[0],
		Aliases:     make([]string, 0),
		Arguments:   orderedmap.New(),
		Description: description,
		Group:       group,
		Public:      public,
		IsTyping:    false,
		Name:        name,
		IsParent:    true,
		IsChild:     false,
		IntegrationTypes: []discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationGuildInstall,
		},
		InstallationContexts: []discordgo.InteractionContextType{
			discordgo.InteractionContextGuild,
		},
	}
	cI.Aliases = append(cI.Aliases, name)
	return cI
}

// Sets the parent properties
func (cI *CommandInfo) SetParent(isParent bool, parentID string) *CommandInfo {
	if !isParent {
		cI.IsChild = true
	}
	cI.IsParent = isParent
	cI.ParentID = parentID
	return cI
}

// AddCmdAlias
// Adds a list of strings as aliases for the command
func (cI *CommandInfo) AddCmdAlias(aliases []string) *CommandInfo {
	if len(aliases) < 1 {
		return cI
	}
	cI.Aliases = aliases
	return cI
}

// AddArg
// Adds an arg to the CommandInfo
func (cI *CommandInfo) AddArg(argument string, typeGuard ArgTypeGuards, match ArgTypes, description string, required bool) *CommandInfo {
	cI.Arguments.Set(argument, &ArgInfo{
		TypeGuard:     typeGuard,
		Description:   description,
		Required:      required,
		Match:         match,
		DefaultOption: "",
		Choices:       make([]*discordgo.ApplicationCommandOptionChoice, 0),
		Regex:         nil,
		AutoComplete:  false,
	})
	return cI
}

// AddFlagArg
// Adds a flag arg, which is a special type of argument
// This type of argument allows for the user to place the "phrase" (e.g: --debug) anywhere
// in the command string and the parser will find it.
func (cI *CommandInfo) AddFlagArg(flag string, typeGuard ArgTypeGuards, match ArgTypes, description string, required bool, defaultOption string) *CommandInfo {
	regexString := flag
	if match == ArgOption {
		// Currently, it only supports a limited character set.
		// todo figure out how to detect any character
		regexString = fmt.Sprintf("--%s (([a-zA-Z0-9:/.]+)|(\"[a-zA-Z0-9:/. ]+\"))", flag)
	} else {
		regexString = fmt.Sprintf("--%s", flag)
	}
	regex, err := regexp2.Compile(regexString, 0)
	if err != nil {
		log.Fatalf("Unable to create regex for flag on command %s flag: %s", cI.Name, flag)
	}
	cI.Arguments.Set(flag, &ArgInfo{
		Description:   description,
		Required:      required,
		Flag:          true,
		Match:         match,
		TypeGuard:     typeGuard,
		DefaultOption: defaultOption,
		Regex:         regex,
	})
	return cI
}

// AddChoice
// Adds an argument choice
func (cI *CommandInfo) AddChoice(arg string, choice string) *CommandInfo {
	v, ok := cI.Arguments.Get(arg)
	if ok {
		vv := v.(*ArgInfo)
		vv.Choices = append(vv.Choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  choice,
			Value: choice,
		})
		cI.Arguments.Set(arg, vv)
	} else {
		log.Errorf("Unable to get argument %s in AddChoice", arg)
		return cI
	}
	return cI
}

// AddChoices
// Adds SubCmd choices
func (cI *CommandInfo) AddChoices(arg string, choices []string) *CommandInfo {
	v, ok := cI.Arguments.Get(arg)
	if ok {
		vv := v.(*ArgInfo)
		optionChoice := make([]*discordgo.ApplicationCommandOptionChoice, 0)
		for _, v := range choices {
			optionChoice = append(optionChoice, &discordgo.ApplicationCommandOptionChoice{
				Name:  v,
				Value: v,
			})
		}
		vv.Choices = append(vv.Choices, optionChoice...)
		cI.Arguments.Set(arg, vv)
	} else {
		log.Errorf("Unable to get argument %s in AddChoices", arg)
		return cI
	}
	return cI
}

// AddChoices
// Adds SubCmd choices
func (cI *CommandInfo) AddChoicesManual(arg string, choices []*discordgo.ApplicationCommandOptionChoice) *CommandInfo {
	v, ok := cI.Arguments.Get(arg)
	if ok {
		vv := v.(*ArgInfo)
		vv.Choices = append(vv.Choices, choices...)
		cI.Arguments.Set(arg, vv)
	} else {
		log.Errorf("Unable to get argument %s in AddChoices", arg)
		return cI
	}
	return cI
}

func (cI *CommandInfo) SetTyping(isTyping bool) *CommandInfo {
	cI.IsTyping = isTyping
	return cI
}

func (cI *CommandInfo) SetAutocomplete(arg string, autocomplete bool) *CommandInfo {
	v, ok := cI.Arguments.Get(arg)
	if ok {
		vv := v.(*ArgInfo)
		vv.AutoComplete = autocomplete
		cI.Arguments.Set(arg, vv)
	} else {
		log.Errorf("Unable to get argument %s in SetAutocomplete", arg)
		return cI
	}
	return cI
}

func (cI *CommandInfo) SetIntegrationType(integrationType ...discordgo.ApplicationIntegrationType) *CommandInfo {
	cI.IntegrationTypes = integrationType
	return cI
}

func (cI *CommandInfo) SetInstallationContext(installationContext ...discordgo.InteractionContextType) *CommandInfo {
	cI.InstallationContexts = installationContext
	return cI
}

// -- Argument Parser --

// ParseArguments
// Version two of the argument parser
func ParseArguments(args string, infoArgs *orderedmap.OrderedMap) *Arguments {
	ar := make(Arguments)

	if args == "" || len(infoArgs.Keys()) < 1 {
		return &ar
	}
	// Split string on spaces to get every "phrase"

	// bool to parse content strings
	moreContent := false
	// Keys of infoArgs
	k := infoArgs.Keys()
	var modK []string
	// First find all flags in the string.
	splitString, ar, modK := findAllFlags(args, k, infoArgs, &ar)
	// Find all the option args (e.g. single 'phrases' or quoted strings)
	// Then return the currentPos, so we can index k and find remaining keys.
	// Also return a modified Arguments struct

	ar, moreContent, splitString, modK = findAllOptionArgs(splitString, modK, infoArgs, &ar)

	// If there is more content, lets find it
	if moreContent == true {
		v, ok := infoArgs.Get(modK[0])
		if !ok {
			return &ar
		}
		vv := v.(*ArgInfo)
		commandContent, _ := createContentString(splitString, 0)
		ar[modK[0]] = CommandArg{
			info:  *vv,
			Value: commandContent,
		}
		return &ar
		// Else return the args struct
	} else {
		return &ar
	}
}

// AddCommand
// Add a command to the bot
func AddCommand(info *CommandInfo, function BotFunction) {
	switch info.Type {
	case ChatCommand:
		AddChatCommand(info, function)
	case UserCommand, MessageCommand:
		AddContextCommand(info, function)
	}
}

// AddChatCommand
// Add a chat command to the bot
func AddChatCommand(info *CommandInfo, function BotFunction) {
	// Build a Command object for this command
	appCommand := createApplicationChatCommand(info)
	command := Command{
		Info:               info,
		Handlers:           make(map[string]BotFunction),
		ApplicationCommand: appCommand,
	}

	command.Handlers["default"] = function

	// adds a alias to a map; command aliases are case-sensitive
	for _, alias := range info.Aliases {
		if _, ok := commandAliases[alias]; ok {
			log.Errorf("Alias was already registered %s for command %s", alias, info.Name)
			continue
		}
		alias = strings.ToLower(alias)
		commandAliases[alias] = info.Name
	}
	// Add the command to the map; command triggers are case-insensitive
	commands[strings.ToLower(info.Name)] = &command
}

// AddContextCommand
// Add a context command to the bot
func AddContextCommand(info *CommandInfo, function BotFunction) {
	appCommand := createApplicationContextCommand(info)
	// Build a Command object for this command
	command := Command{
		Info:               info,
		Handlers:           make(map[string]BotFunction),
		ApplicationCommand: appCommand,
	}

	command.Handlers["default"] = function

	// Add the command to the map; command triggers are case-insensitive
	commands[strings.ToLower(info.Name)] = &command
}

// AddCommandHandler
// Adds a command handler to the bot
func AddCommandHandler(info *CommandInfo, function BotFunction, handler string) {
	if _, ok := commands[strings.ToLower(info.Name)]; !ok {
		log.Errorf("Command was not found")
		return
	}
	commands[strings.ToLower(info.Name)].Handlers[handler] = function
}

// AddAutoCompleteHandler
// Adds an autocomplete handler to the bot
func AddAutoCompleteHandler(info *CommandInfo, function BotFunction, handler string) {
	if _, ok := commands[strings.ToLower(info.Name)]; !ok {
		log.Errorf("Command was not found")
		return
	}
	commands[info.Name].Handlers[fmt.Sprintf("ac:%s", strings.ToLower(handler))] = function
}

// AddComponentHandler
// Adds a component handler to the bot
func AddComponentHandler(handler string, function BotFunction) {
	if _, ok := componentHandlers[handler]; ok {
		log.Errorf("Component handler was already registered %s", handler)
		return
	}
	componentHandlers[handler] = function
}

// RegisterSlashCommands
// Defaults to adding Global slash commands
// Currently hard coded to guild commands for testing
func RegisterSlashCommands(guildId string, c chan string) {
	for _, v := range commands {
		_, err := Session.ApplicationCommandCreate(Session.State.User.ID, guildId, v.ApplicationCommand)
		if err != nil {
			c <- "Unable to register slash commands :/"
			log.Errorf("Cannot create '%v' command: %v", v.Info, err)
			log.Errorf("%v", v.ApplicationCommand)
			return
		}
	}
	c <- "Finished registering slash commands"
}

// GetCommands
// Provide a way to read commands without making it possible to modify their functions
func GetCommands() map[string]CommandInfo {
	list := make(map[string]CommandInfo)
	for x, y := range commands {
		list[x] = *y.Info
	}
	return list
}

// SendAutocompleteChoices
// Sends the choices to the user
func (ctx *Context) SendAutocompleteChoices(choices []*discordgo.ApplicationCommandOptionChoice) {
	err := Session.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		log.Errorf("Error sending autocomplete choices %s", err)
	}
}

// commandHandler
// This handler will be added to a *discordgo.Session, and will scan an incoming messages for commands to run
func commandHandler(session *discordgo.Session, message *discordgo.MessageCreate) {
	// Try getting an object for the current channel, with a fallback in case session.state is not ready or is nil
	channel, err := session.State.Channel(message.ChannelID)
	if err != nil {
		if channel, err = session.Channel(message.ChannelID); err != nil {
			return
		}
	}

	// Ignore messages sent by the bot
	if message.Author.ID == session.State.User.ID {
		return
	}

	g := getGuild(message.GuildID)

	trigger, argString := ExtractCommand(&g.Info, message.Content)
	if trigger == nil {
		return
	}
	// Only do further checks if the user is not a bot admin
	if !IsAdmin(message.Author.ID) {
		// Ignore the command if it is globally disabled
		if g.IsGloballyDisabled(*trigger) {
			return
		}

		// Ignore the command if this channel has blocked the command
		if g.CommandIsDisabledInChannel(*trigger, message.ChannelID) {
			return
		}

		// Ignore any message if the user is banned from using the bot
		if !g.MemberOrRoleIsWhitelisted(message.Author.ID) || g.MemberOrRoleIsIgnored(message.Author.ID) {
			return
		}

		// Ignore the message if this channel is not whitelisted, or if it is ignored
		if !g.ChannelIsWhitelisted(message.ChannelID) || g.ChannelIsIgnored(message.ChannelID) {
			return
		}
	}

	//Get the command to run
	// Error Checking
	command, ok := commands[commandAliases[*trigger]]
	if !ok {
		log.Errorf("Command was not found")
		return
	}
	// Check if the command is public, or if the current user is a bot moderator
	// Bot admins supercede both checks
	if IsAdmin(message.Author.ID) || command.Info.Public || g.IsMod(message.Author.ID) {
		// Run the command with the necessary context
		if command.Info.IsTyping && g.Info.ResponseChannelId == "" {
			_ = Session.ChannelTyping(message.ChannelID)
		}
		// The command is valid, so now we need to delete the invoking message if that is configured
		if g.Info.DeletePolicy {
			err := Session.ChannelMessageDelete(message.ChannelID, message.ID)
			if err != nil {
				SendErrorReport(message.GuildID, message.ChannelID, message.Author.ID, "Failed to delete message: "+message.ID, err)
			}
		}

		defer handleCommandError(g.ID, channel.ID, message.Author.ID)
		if command.Info.IsParent {
			// handleChildCommand(*argString, command, message.Message, g)
			return
		}
		command.Handlers["default"](&Context{
			Guild:   g,
			Cmd:     *command.Info,
			Args:    *ParseArguments(*argString, command.Info.Arguments),
			Message: message.Message,
		})
		// Makes sure that variables ran in ParseArguments are gone.
		if commandsGC == 25 && commandsGC > 25 {
			debug.FreeOSMemory()
			commandsGC = 0
		} else {
			commandsGC++
		}
		return
	}

}

func handleCommandError(gID string, cId string, uId string) {
	if r := recover(); r != nil {
		log.Warningf("Recovering from panic: %s", r)
		log.Warningf("Sending Error report to admins")
		SendErrorReport(gID, cId, uId, "Error!", r.(runtime.Error))
		message, err := Session.ChannelMessageSend(cId, "Error!")
		if err != nil {
			log.Errorf("err sending message %s", err)
		}
		time.Sleep(5 * time.Second)
		_ = Session.ChannelMessageDelete(cId, message.ID)
		return
	}
}
