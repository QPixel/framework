package framework

import (
	"github.com/bwmarrin/discordgo"
	"runtime"
)

// -- Types and Structs --

// slashCommandTypes
// A map of *short hand* slash commands types to their discordgo counterparts
// TODO move this over to interaction.go
var slashCommandTypes = map[ArgTypeGuards]discordgo.ApplicationCommandOptionType{
	Int:     discordgo.ApplicationCommandOptionInteger,
	String:  discordgo.ApplicationCommandOptionString,
	Channel: discordgo.ApplicationCommandOptionChannel,
	User:    discordgo.ApplicationCommandOptionUser,
	Role:    discordgo.ApplicationCommandOptionRole,
	Boolean: discordgo.ApplicationCommandOptionBoolean,
	//SubCmd:    discordgo.ApplicationCommandOptionSubCommand,
	//SubCmdGrp: discordgo.ApplicationCommandOptionSubCommandGroup,
}

// getSlashCommandStruct
// Creates a slash command struct
// todo work on sub command stuff
func createSlashCommandStruct(info *CommandInfo) (st *discordgo.ApplicationCommand) {
	if info.Arguments == nil || len(info.Arguments.Keys()) < 1 {
		st = &discordgo.ApplicationCommand{
			Name:        info.Trigger,
			Description: info.Description,
		}
		return
	}
	st = &discordgo.ApplicationCommand{
		Name:        info.Trigger,
		Description: info.Description,
		Options:     make([]*discordgo.ApplicationCommandOption, len(info.Arguments.Keys())),
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
			Type:        sType,
			Name:        k,
			Description: vv.Description,
			Required:    vv.Required,
		}
		if vv.Choices != nil {
			optionStruct.Choices = make([]*discordgo.ApplicationCommandOptionChoice, len(vv.Choices))
			for i, k := range vv.Choices {
				optionStruct.Choices[i] = &discordgo.ApplicationCommandOptionChoice{
					Name:  k,
					Value: k,
				}
			}
		}
		st.Options[i] = &optionStruct
	}
	return
}

// Creates a slash subcmd struct
func createSlashSubCmdStruct(info *CommandInfo, childCmds map[string]Command) (st *discordgo.ApplicationCommand) {
	st = &discordgo.ApplicationCommand{
		Name:        info.Trigger,
		Description: info.Description,
		Options:     make([]*discordgo.ApplicationCommandOption, len(childCmds)),
	}
	currentPos := 0
	for _, v := range childCmds {
		// Stupid inline thing
		if ar, _ := v.Info.Arguments.Get(v.Info.Arguments.Keys()[0]); ar.(*ArgInfo).TypeGuard == SubCmdGrp {

		} else {
			//Pixel:
			//Yes I know this is O(N^2). Most likely I could get something better
			//todo: refactor so this isn't as bad for performance
			st.Options[currentPos] = v.Info.CreateAppOptSt()
			currentPos++
		}
	}
	return st
}

// -- Interaction Handlers --

// handleInteraction
// Handles a slash command interaction.
func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleInteractionCommand(s, i)
		break
	case discordgo.InteractionMessageComponent:
		handleMessageComponents(s, i)
	}
	return
}

// handleInteractionCommand
// Handles a slash command
func handleInteractionCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	g := getGuild(i.GuildID)

	trigger := i.ApplicationCommandData().Name
	if !IsAdmin(i.Member.User.ID) {
		// Ignore the command if it is globally disabled
		if g.IsGloballyDisabled(trigger) {
			ErrorResponse(i.Interaction, "Command is globally disabled", trigger)
			return
		}

		// Ignore the command if this channel has blocked the command
		if g.CommandIsDisabledInChannel(trigger, i.ChannelID) {
			ErrorResponse(i.Interaction, "Command is disabled in this channel!", trigger)
			return
		}

		// Ignore any message if the user is banned from using the bot
		if !g.MemberOrRoleIsWhitelisted(i.Member.User.ID) || g.MemberOrRoleIsIgnored(i.Member.User.ID) {
			return
		}

		// Ignore the message if this channel is not whitelisted, or if it is ignored
		if !g.ChannelIsWhitelisted(i.ChannelID) || g.ChannelIsIgnored(i.ChannelID) {
			return
		}
	}

	command := commands[trigger]
	if IsAdmin(i.Member.User.ID) || command.Info.Public || g.IsMod(i.Member.User.ID) {
		// Check if the command is public, or if the current user is a bot moderator
		// Bot admins supercede both checks

		defer handleSlashCommandError(*i.Interaction)
		command.Function(&Context{
			Guild:       g,
			Cmd:         command.Info,
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
		return
	}
}

func handleMessageComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	content := "Currently testing customid " + i.MessageComponentData().CustomID
	i.Message.Embeds[0].Description = content
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		// Buttons also may update the message which they was attached to.
		// Or may just acknowledge (InteractionResponseDeferredMessageUpdate) that the event was received and not update the message.
		// To update it later you need to use interaction response edit endpoint.
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			TTS:    false,
			Embeds: i.Message.Embeds,
		},
	})
	return
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
		log.Warningf("Recovering from panic: %s", r)
		log.Warningf("Sending Error report to admins")
		SendErrorReport(i.GuildID, i.ChannelID, i.Member.User.ID, "Error!", r.(runtime.Error))
		message, err := Session.InteractionResponseEdit(Session.State.User.ID, &i, &discordgo.WebhookEdit{
			Content: "error executing command",
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
		return
	}
	return
}
