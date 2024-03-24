package framework

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
	errors "gitlab.com/tozd/go/errors"
)

// -- Types and Structs --

// slashCommandTypes
// A map of *short hand* slash commands types to their discordgo counterparts
var slashCommandTypes = map[ArgTypeGuards]discordgo.ApplicationCommandOptionType{
	Int:       discordgo.ApplicationCommandOptionInteger,
	String:    discordgo.ApplicationCommandOptionString,
	Channel:   discordgo.ApplicationCommandOptionChannel,
	User:      discordgo.ApplicationCommandOptionUser,
	Role:      discordgo.ApplicationCommandOptionRole,
	Boolean:   discordgo.ApplicationCommandOptionBoolean,
	SubCmd:    discordgo.ApplicationCommandOptionSubCommand,
	SubCmdGrp: discordgo.ApplicationCommandOptionSubCommandGroup,
}

var genericError = "error executing command"

func createApplicationChatCommand(info *CommandInfo) (st *discordgo.ApplicationCommand) {
	if info.Arguments == nil || len(info.Arguments.Keys()) < 1 {
		st = &discordgo.ApplicationCommand{
			Name:             info.Name,
			Description:      info.Description,
			Type:             discordgo.ChatApplicationCommand,
			IntegrationTypes: &info.IntegrationTypes,
			Contexts:         &info.InstallationContexts,
		}
		return
	}
	st = &discordgo.ApplicationCommand{
		Name:             info.Name,
		Description:      info.Description,
		Options:          make([]*discordgo.ApplicationCommandOption, len(info.Arguments.Keys())),
		Type:             discordgo.ChatApplicationCommand,
		IntegrationTypes: &info.IntegrationTypes,
		Contexts:         &info.InstallationContexts,
	}
	for i, k := range info.Arguments.Keys() {
		v, _ := info.Arguments.Get(k)
		vv := v.(*ArgInfo)
		var sType discordgo.ApplicationCommandOptionType
		if val, ok := slashCommandTypes[vv.TypeGuard]; ok {
			sType = val
		} else {
			sType = slashCommandTypes["String"]
		}
		optionStruct := discordgo.ApplicationCommandOption{
			Type:         sType,
			Name:         k,
			Description:  vv.Description,
			Required:     vv.Required,
			Autocomplete: vv.AutoComplete,
		}
		if len(vv.Choices) > 0 {
			optionStruct.Choices = vv.Choices
		}
		st.Options[i] = &optionStruct
	}
	return
}

// func createApplicationContextCommand(info *CommandInfo, context_type discordgo.ApplicationCommandType) (st *discordgo.ApplicationCommand) {

// }

// -- Interaction Handlers --

// handleInteraction
// Handles a slash command interaction.
func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleInteractionCommand(s, i)
	case discordgo.InteractionMessageComponent:
		handleMessageComponents(s, i)
	case discordgo.InteractionApplicationCommandAutocomplete:
		handleAutoComplete(i)
	}
}

// handleInteractionCommand
// Handles a slash command
func handleInteractionCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Let's check if this is a user command, if so lets handle it separately
	if i.Interaction.Member == nil && i.Interaction.GuildID == "" {
		handleUserApplicationCommand(s, i)
		return
	}

	g := getGuild(i.GuildID)

	trigger := i.ApplicationCommandData().Name
	log.Debugf("Handling command %s", trigger)
	// if !IsAdmin(i.Member.User.ID) {
	// 	// Ignore the command if it is globally disabled
	// 	if g.IsGloballyDisabled(trigger) {
	// 		ErrorResponse(i.Interaction, "Command is globally disabled", trigger)
	// 		return
	// 	}

	// 	// Ignore the command if this channel has blocked the command
	// 	if g.CommandIsDisabledInChannel(trigger, i.ChannelID) {
	// 		ErrorResponse(i.Interaction, "Command is disabled in this channel!", trigger)
	// 		return
	// 	}

	// 	// Ignore any message if the user is banned from using the bot
	// 	if !g.MemberOrRoleIsWhitelisted(i.Member.User.ID) || g.MemberOrRoleIsIgnored(i.Member.User.ID) {
	// 		return
	// 	}

	// 	// Ignore the message if this channel is not whitelisted, or if it is ignored
	// 	if !g.ChannelIsWhitelisted(i.ChannelID) || g.ChannelIsIgnored(i.ChannelID) {
	// 		return
	// 	}
	// }

	command := commands[trigger]
	log.Debugf("Command %s found %#v", trigger, command)
	// if IsAdmin(i.Member.User.ID) || command.Info.Public || g.IsMod(i.Member.User.ID) {
	// Check if the command is public, or if the current user is a bot moderator
	// Bot admins supercede both checks
	// }
	log.Debugf("%#v", i.Interaction)
	defer handleSlashCommandError(*i.Interaction)
	command.Handlers["default"](&Context{
		Guild:       g,
		Cmd:         *command.Info,
		Args:        *ParseInteractionArgs(i.ApplicationCommandData().Options),
		Interaction: i.Interaction,
		Message: &discordgo.Message{
			Member:    i.Member,
			Author:    i.Member.User,
			ChannelID: i.ChannelID,
			GuildID:   i.GuildID,
			Content:   "",
		},
	})
}

func handleUserApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	trigger := i.ApplicationCommandData().Name
	log.Debugf("Handling user command %s", trigger)
	command := commands[trigger]
	log.Debugf("Command %s found %#v", trigger, command)
	defer handleSlashCommandError(*i.Interaction)
	command.Handlers["default"](&Context{
		Guild:       nil,
		Cmd:         *command.Info,
		Args:        *ParseInteractionArgs(i.ApplicationCommandData().Options),
		Interaction: i.Interaction,
		Message: &discordgo.Message{
			Member: &discordgo.Member{
				User: i.User,
			},
			Author:    i.User,
			ChannelID: i.ChannelID,
			Content:   "",
		},
	})

}

func handleMessageComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	componentName := i.MessageComponentData().CustomID
	if _, ok := componentHandlers[componentName]; !ok {
		log.Errorf("No component found for %s", componentName)
		return
	}

	defer handleSlashCommandError(*i.Interaction)
	componentHandlers[componentName](&Context{
		Guild:       getGuild(i.GuildID),
		Cmd:         CommandInfo{},
		Args:        map[string]CommandArg{},
		Interaction: i.Interaction,
		Message: &discordgo.Message{
			Member:    i.Member,
			Author:    i.Member.User,
			ChannelID: i.ChannelID,
			GuildID:   i.GuildID,
			Content:   "",
		},
	})
}

func handleAutoComplete(i *discordgo.InteractionCreate) {
	commandName := i.ApplicationCommandData().Name
	for _, option := range i.ApplicationCommandData().Options {
		if option.Focused {
			command := commands[strings.ToLower(commandName)]
			if command == nil {
				log.Errorf("No command found for autocomplete %s", commandName)
				return
			}

			// All AutoComplete handlers are prefixed with "ac:"
			handler := command.Handlers[fmt.Sprintf("ac:%s", strings.ToLower(option.Name))]

			if handler == nil {
				log.Errorf("No handler found for autocomplete %s", commandName)
				return
			}

			defer handleAutoCompleteError(*i.Interaction, "Error executing autocomplete")

			handler(&Context{
				Guild:       getGuild(i.GuildID),
				Cmd:         *command.Info,
				Args:        *ParseInteractionArgs(i.ApplicationCommandData().Options),
				Interaction: i.Interaction,
			})
		}
	}

}

// -- Slash Argument Parsing Helpers --

// ParseInteractionArgs
// Parses Interaction args
func ParseInteractionArgs(options []*discordgo.ApplicationCommandInteractionDataOption) *map[string]CommandArg {
	var args = make(map[string]CommandArg)
	for _, v := range options {
		args[v.Name] = CommandArg{
			info:  ArgInfo{},
			Value: v.Value,
		}
		if v.Options != nil {
			ParseInteractionArgsR(v.Options, &args)
		}
	}
	return &args
}

// ParseInteractionArgsR
// Parses interaction args recursively
func ParseInteractionArgsR(options []*discordgo.ApplicationCommandInteractionDataOption, args *map[string]CommandArg) {
	for _, v := range options {
		(*args)[v.Name] = CommandArg{
			info:  ArgInfo{},
			Value: v.StringValue(),
		}
		if v.Options != nil {
			ParseInteractionArgsR(v.Options, *&args)
		}
	}
}

// -- :shrug: --

// RemoveGuildSlashCommands
// Removes all guild slash commands.
func RemoveGuildSlashCommands(guildID string) {
	commands, err := Session.ApplicationCommands(Session.State.User.ID, guildID)
	if err != nil {
		log.Errorf("Error getting all slash commands %s", err)
		return
	}
	for _, k := range commands {
		err = Session.ApplicationCommandDelete(Session.State.User.ID, guildID, k.ID)
		if err != nil {
			log.Errorf("error deleting slash command %s %s %s", k.Name, k.ID, err)
			continue
		}
	}
}

func handleSlashCommandError(i discordgo.Interaction) {
	if r := recover(); r != nil {
		e := errors.WithStack(r.(error))
		log.Warningf("Recovering from panic: %s", e)
		log.Warningf("Sending Error report to admins")
		SendErrorReport(i.GuildID, i.ChannelID, i.Member.User.ID, "Error!", e)
		message, err := Session.InteractionResponseEdit(&i, &discordgo.WebhookEdit{
			Content: &genericError,
		})
		if err != nil {
			Session.InteractionRespond(&i, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   1 << 6,
					Content: "error executing command",
				},
			})
			log.Errorf("err sending message %s", err)
		}
		Session.ChannelMessageDelete(i.ChannelID, message.ID)
	}
}

func handleAutoCompleteError(i discordgo.Interaction, message string) {
	if r := recover(); r != nil {
		log.Warningf("Recovering from panic: %s", r)
		log.Warningf("Sending Error report to admins")
		SendErrorReport(i.GuildID, i.ChannelID, i.Member.User.ID, "Error!", r.(runtime.Error))
	}
}
