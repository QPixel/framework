package core

import (
	"github.com/QPixel/orderedmap"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// TODO Clean up this file
// commands.go
// This file contains everything required to add core commands to the bot, and parse commands from a message

// GroupTypes
const (
	Moderation = "moderation"
	Module     = "module"
	Utility    = "utility"
)

// CommandInfo
// The definition of a command's info. This is everything about the command, besides the function it will run
type CommandInfo struct {
	Aliases     []string               // Aliases for the normal trigger
	Arguments   *orderedmap.OrderedMap // Arguments for the command
	Description string                 // A short description of what the command does
	Group       string                 // The group this command belongs to
	ParentID    string                 // The ID of the parent command
	Public      bool                   // Whether or not non-admins and non-mods can use this command
	IsTyping    bool                   // Whether or not the command will show a typing thing when ran.
	IsParent    bool                   // If the command is the parent of a subcommand tree
	IsChild     bool                   // If the command is the child
	Trigger     string                 // The string that will trigger the command
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
// Contexts are also passed as pointers so they are not re-allocated when passed through
type BotFunction func(ctx *Context)

// Command
// The definition of a command, which is that command's information, along with the function it will run
type Command struct {
	Info     CommandInfo
	Function BotFunction
}

type ChildCommand map[string]map[string]Command

// CustomCommand
// A type that defines a custom command
type CustomCommand struct {
	Content     string // The content of the custom command. Custom commands are just special strings after all
	InvokeCount int64  // How many times the command has been invoked; int64 for easier use with json
	Public      bool   // Whether or not non-admins and non-mods can use this command
}

// commands
// All of the registered core commands (not custom commands)
// This is private so that other commands cannot modify it
var commands = make(map[string]Command)

// childCommands
// All of the registered childcommands (subcmdgrps)
// This is private so other commands cannot modify it
var childCommands = make(ChildCommand)

// Command Aliases
// A map of aliases to command triggers
var commandAliases = make(map[string]string)

// slashCommands
// All of the registered core commands that are also slash commands
// This is also private so other commands cannot modify it
var slashCommands = make(map[string]discordgo.ApplicationCommand)

// AddCommand
// Add a command to the bot
func AddCommand(info *CommandInfo, function BotFunction) {
	// Add Trigger to the alias
	info.Aliases = append(info.Aliases, info.Trigger)
	// Build a Command object for this command
	command := Command{
		Info:     *info,
		Function: function,
	}
	// adds a alias to a map; command aliases are case-sensitive
	for _, alias := range info.Aliases {
		if _, ok := commandAliases[alias]; ok {
			log.Errorf("Alias was already registered %s for command %s", alias, info.Trigger)
			continue
		}
		alias = strings.ToLower(alias)
		commandAliases[alias] = info.Trigger
	}
	// Add the command to the map; command triggers are case-insensitive
	commands[strings.ToLower(info.Trigger)] = command
}

// AddChildCommand
// Adds a child command to the bot.
func AddChildCommand(info *CommandInfo, function BotFunction) {
	// Build a Command object for this command
	command := Command{
		Info:     *info,
		Function: function,
	}
	parentID := strings.ToLower(info.ParentID)
	if childCommands[parentID] == nil {
		childCommands[parentID] = make(map[string]Command)
	}
	// Add the command to the map; command triggers are case-insensitive
	childCommands[parentID][command.Info.Trigger] = command
}

// AddSlashCommand
// Adds a slash command to the bot
// Allows for separation between normal commands and slash commands
func AddSlashCommand(info *CommandInfo) {
	s := createSlashCommandStruct(info)
	slashCommands[strings.ToLower(info.Trigger)] = *s
}

// AddSlashCommands
// Defaults to adding Global slash commands
// Currently hard coded to guild commands for testing
func AddSlashCommands(guildId string, c chan string) {
	for _, v := range slashCommands {
		_, err := Session.ApplicationCommandCreate(Session.State.User.ID, guildId, &v)
		if err != nil {
			c <- "Unable to register slash commands :/"
			log.Errorf("Cannot create '%v' command: %v", v.Name, err)
			log.Errorf("%s", v.Options)
			return
		}
	}
	c <- "Finished registering slash commands"
	return
}

// GetCommands
// Provide a way to read commands without making it possible to modify their functions
func GetCommands() map[string]CommandInfo {
	list := make(map[string]CommandInfo)
	for x, y := range commands {
		list[x] = y.Info
	}
	return list
}

// customCommandHandler
// Given a custom command, interpret and run it
func customCommandHandler(command CustomCommand, args []string, message *discordgo.Message) {
	//TODO
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

	// If we are in DMs, ignore the message
	// In the future, this can be used to handle special DM-only commands
	if channel.Type == discordgo.ChannelTypeDM {
		return
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
	isCustom := false
	if _, ok := commands[commandAliases[*trigger]]; !ok {
		if !g.IsCustomCommand(*trigger) {
			return
		} else {
			isCustom = true
		}
	}
	// Only do further checks if the user is not a bot admin
	if !IsAdmin(message.Author.ID) {
		// Ignore the command if it is globally disabled
		if g.IsGloballyDisabled(*trigger) {
			return
		}

		// Ignore the command if this channel has blocked the trigger
		if g.TriggerIsDisabledInChannel(*trigger, message.ChannelID) {
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

	// The command is valid, so now we need to delete the invoking message if that is configured
	if g.Info.DeletePolicy {
		err := Session.ChannelMessageDelete(message.ChannelID, message.ID)
		if err != nil {
			SendErrorReport(message.GuildID, message.ChannelID, message.Author.ID, "Failed to delete message: "+message.ID, err)
		}
	}
	if !isCustom {
		//Get the command to run
		// Error Checking
		command, ok := commands[commandAliases[*trigger]]
		if !ok {
			log.Errorf("Command was not found")
			if IsAdmin(message.Author.ID) {
				Session.MessageReactionAdd(message.ChannelID, message.ID, "<:redtick:861413502991073281>")
				Session.ChannelMessageSendReply(message.ChannelID, "<:redtick:861413502991073281> Error! Command not found!", message.MessageReference)
			}
			return
		}
		// Check if the command is public, or if the current user is a bot moderator
		// Bot admins supercede both checks
		if IsAdmin(message.Author.ID) || command.Info.Public || g.IsMod(message.Author.ID) {
			// Run the command with the necessary context
			if command.Info.IsTyping && g.Info.ResponseChannelId == "" {
				_ = Session.ChannelTyping(message.ChannelID)
			}
			if command.Info.IsParent {
				handleChildCommand(*argString, command, message.Message, g)
				return
			}
			command.Function(&Context{
				Guild:   g,
				Cmd:     command.Info,
				Args:    *ParseArguments(*argString, command.Info.Arguments),
				Message: message.Message,
			})
			return
		}
	}
}

// -- Helper Methods
func handleChildCommand(argString string, command Command, message *discordgo.Message, g *Guild) {
	split := strings.SplitN(argString, " ", 2)
	// First lets see if this subcmd even exists
	v, ok := command.Info.Arguments.Get("subcmdgrp")
	// the command doesn't even have a subcmdgrp arg, return
	if !ok {
		return
	}

	choices := v.(*ArgInfo).Choices
	subCmdExist := false
	for _, choice := range choices {
		if split[0] != choice {
			continue
		} else {
			subCmdExist = true
			break
		}
	}
	if !subCmdExist {
		command.Function(&Context{
			Guild:   g,
			Cmd:     command.Info,
			Args:    nil,
			Message: message,
		})
		return
	}
	childCmd, ok := childCommands[command.Info.Trigger][split[0]]
	if !ok || len(split) < 2 {
		command.Function(&Context{
			Guild:   g,
			Cmd:     command.Info,
			Args:    nil,
			Message: message,
		})
		return
	}
	childCmd.Function(&Context{
		Guild:   g,
		Cmd:     childCmd.Info,
		Args:    *ParseArguments(split[1], childCmd.Info.Arguments),
		Message: message,
	})
	return
}
