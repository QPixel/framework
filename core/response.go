package core

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// response.go
// This file contains structures and functions that make it easier to create and send response embeds

// ResponseComponents
// Stores the components for response
// allows for functions to add data
type ResponseComponents struct {
	Components        []discordgo.MessageComponent
	SelectMenuOptions []discordgo.SelectMenuOption
}

// Response
// The Response type, can be build and sent to a given guild
type Response struct {
	Ctx                *Context
	Success            bool
	Loading            bool
	Ephemeral          bool
	Embed              *discordgo.MessageEmbed
	ResponseComponents *ResponseComponents
}

// CreateField
// Create message field to use for an embed
func CreateField(name string, value string, inline bool) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}

// CreateEmbed
// Create an embed
func CreateEmbed(color int, title string, description string, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Fields:      fields,
	}
}

// CreateComponentFields
// Returns a slice of a Message Component, containing a singular ActionsRow
func CreateComponentFields() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{},
	}
}

// NewResponse
// Create a response object for a guild, which starts off as an empty Embed which will have fields added to it
// The response starts with some "auditing" information
// The embed will be finalized in .Send()
func NewResponse(ctx *Context, messageComponents bool, ephemeral bool) *Response {
	r := &Response{
		Ctx:   ctx,
		Embed: CreateEmbed(0, "", "", nil),
		ResponseComponents: &ResponseComponents{
			Components:        nil,
			SelectMenuOptions: nil,
		},
		Loading:   ctx.Cmd.IsTyping,
		Ephemeral: ephemeral,
	}
	if messageComponents {
		r.ResponseComponents.Components = CreateComponentFields()
		r.ResponseComponents.SelectMenuOptions = []discordgo.SelectMenuOption{}
	}
	if r.Loading && ctx.Interaction != nil {
		if ephemeral {
			_ = Session.InteractionRespond(r.Ctx.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					// Ephemeral is type 64 don't ask why
					Flags: 1 << 6,
				},
			})
		} else {
			_ = Session.InteractionRespond(r.Ctx.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
		}

	}
	// If the command context is not empty, append the command
	if ctx.Cmd.Trigger != "" {
		// Get the command used as a string, and all interpreted arguments, so it can be a part of the output
		commandUsed := ""
		if r.Ctx.Cmd.IsChild {
			commandUsed = fmt.Sprintf("%s%s %s", r.Ctx.Guild.Info.Prefix, r.Ctx.Cmd.ParentID, r.Ctx.Cmd.Trigger)
		} else {
			commandUsed = r.Ctx.Guild.Info.Prefix + r.Ctx.Cmd.Trigger
		}
		// Just makes the thing prettier
		if ctx.Interaction != nil {
			commandUsed = "/" + r.Ctx.Cmd.Trigger
		}
		for _, k := range r.Ctx.Cmd.Arguments.Keys() {
			arg := ctx.Args[k]
			if arg.StringValue() == "" {
				continue
			}
			vv, ok := r.Ctx.Cmd.Arguments.Get(k)

			if ok {
				argInfo := vv.(*ArgInfo)
				switch argInfo.TypeGuard {
				case Int:
					fallthrough
				case Boolean:
					fallthrough
				case String:
					commandUsed += " " + arg.StringValue()
					break
				case User:
					user, err := arg.UserValue(Session)
					if err != nil {
						commandUsed += " " + arg.StringValue()
					} else {
						commandUsed += " " + user.Mention()
					}
				case Role:
					role, err := arg.RoleValue(Session, r.Ctx.Guild.ID)
					if err != nil {
						commandUsed += " " + arg.StringValue()
					} else {
						commandUsed += " " + role.Mention()
					}
				case Channel:
					channel, err := arg.ChannelValue(Session)
					if err != nil {
						commandUsed += " " + arg.StringValue()
					} else {
						commandUsed += " " + channel.Mention()
					}
				}
			} else {
				commandUsed += " " + arg.StringValue()
			}
		}

		commandUsed = "```\n" + commandUsed + "\n```"

		r.AppendField("Command used:", commandUsed, false)
	}

	// If the message is not nil, append an invoker
	if ctx.Message != nil {
		r.AppendField("Invoked by:", r.Ctx.Message.Author.Mention(), false)
	}

	return r
}

// -- Fields --

// AppendField
// Create a new basic field and append it to an existing Response
func (r *Response) AppendField(name string, value string, inline bool) {
	r.Embed.Fields = append(r.Embed.Fields, CreateField(name, value, inline))
}

// PrependField
// Create a new basic field and prepend it to an existing Response
func (r *Response) PrependField(name string, value string, inline bool) {
	fields := []*discordgo.MessageEmbedField{CreateField(name, value, inline)}
	r.Embed.Fields = append(fields, r.Embed.Fields...)
}

// AppendUsage
// Add the command usage to the response. Intended for syntax error responses
func (r *Response) AppendUsage() {
	if r.Ctx.Cmd.Description == "" {
		r.AppendField("Command description:", "no description", false)
		return
	}
	r.AppendField("Command description:", r.Ctx.Cmd.Description, false)
	//r.AppendField("Command usage:", r.Ctx.Guild.GetCommandUsage(r.Ctx.Cmd), false)

}

// -- Message Components --

func CreateButton(label string, style discordgo.ButtonStyle, customID string, url string, disabled bool) *discordgo.Button {
	button := &discordgo.Button{
		Label:    label,
		Style:    style,
		Disabled: disabled,
		Emoji:    discordgo.ComponentEmoji{},
		URL:      url,
		CustomID: customID,
	}
	return button
}

func CreateDropDown(customID string, placeholder string, options []discordgo.SelectMenuOption) discordgo.SelectMenu {
	dropDown := discordgo.SelectMenu{
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
	return dropDown
}

// AppendButton
// Appends a button
func (r *Response) AppendButton(label string, style discordgo.ButtonStyle, url string, customID string, rowID int) {
	row := r.ResponseComponents.Components[rowID].(discordgo.ActionsRow)
	row.Components = append(row.Components, CreateButton(label, style, customID, url, false))
	r.ResponseComponents.Components[rowID] = row
}

//AppendDropDown
// Adds a DropDown component
func (r *Response) AppendDropDown(customID string, placeholder string, noNewRow bool) {
	if noNewRow {
		row := r.ResponseComponents.Components[0].(discordgo.ActionsRow)
		row.Components = append(row.Components, CreateDropDown(customID, placeholder, r.ResponseComponents.SelectMenuOptions))
		r.ResponseComponents.Components[0] = row
	} else {
		actionRow := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    customID,
					Placeholder: placeholder,
					Options:     r.ResponseComponents.SelectMenuOptions,
				},
			},
		}
		r.ResponseComponents.Components = append(r.ResponseComponents.Components, actionRow)
	}
}

// Send
// Send a compiled response
func (r *Response) Send(success bool, title string, description string) {
	// Determine what color to use based on the success state
	var color int
	if success {
		color = ColorSuccess
	} else {
		// On failure, also append the command usage
		r.AppendUsage()
		color = ColorFailure
	}

	// Fill out the main embed
	r.Embed.Title = title
	r.Embed.Description = description
	r.Embed.Color = color

	// If guild is nil, this is intended to be sent to Bot Admins
	if r.Ctx.Guild == nil {
		for admin := range botAdmins {
			dmChannel, dmCreateErr := Session.UserChannelCreate(admin)
			if dmCreateErr != nil {
				// Since error reports also use DMs, sending this as an error report would be redundant
				// Just log the error
				log.Errorf("Failed sending Response DM to admin: %s; Response title: %s", admin, r.Embed.Title)
				return
			}
			_, dmSendErr := Session.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
				Embed:      r.Embed,
				Components: r.ResponseComponents.Components,
			})
			if dmSendErr != nil {
				// Since error reports also use DMs, sending this as an error report would be redundant
				// Just log the error
				log.Errorf("Failed sending Response DM to admin: %s; Response title: %s", admin, r.Embed.Title)
				return
			}
			return
		}
	}

	// If this is a interaction (slash command)
	// Run it as a interaction response and then return early
	if r.Ctx.Interaction != nil {
		// Some commands take a while to load
		// Slash commands expect a response in 3 seconds or the interaction gets invalidated
		if r.Loading {
			// Check to see if the command is ephemeral (only shown to the user)
			if r.Ephemeral {
				_, err := Session.InteractionResponseEdit(Session.State.User.ID, r.Ctx.Interaction, &discordgo.WebhookEdit{
					Components: r.ResponseComponents.Components,
					Embeds: []*discordgo.MessageEmbed{
						r.Embed,
					},
				})
				// Just in case the interaction gets removed.
				if err != nil {
					if err != nil {
						SendErrorReport(r.Ctx.Guild.ID, r.Ctx.Interaction.ChannelID, r.Ctx.Message.Author.ID, "Unable to send interaction messages", err)
					}
					if r.Ctx.Guild.Info.ResponseChannelId != "" {
						_, err = Session.ChannelMessageSendEmbed(r.Ctx.Guild.Info.ResponseChannelId, r.Embed)

					} else {
						_, err = Session.ChannelMessageSendEmbed(r.Ctx.Message.ChannelID, r.Embed)
					}

					if err != nil {
						SendErrorReport(r.Ctx.Guild.ID, r.Ctx.Interaction.ChannelID, r.Ctx.Message.Author.ID, "Unable to send message", err)
					}
				}
			} else {
				_, err := Session.InteractionResponseEdit(Session.State.User.ID, r.Ctx.Interaction, &discordgo.WebhookEdit{
					Content: "",
					Embeds: []*discordgo.MessageEmbed{
						r.Embed,
					},
					Components: r.ResponseComponents.Components,
				})
				// Just in case the interaction gets removed.
				if err != nil {
					_, err := Session.ChannelMessageSendEmbed(r.Ctx.Guild.Info.ResponseChannelId, r.Embed)
					if err != nil {
						_, err = Session.ChannelMessageSendEmbed(r.Ctx.Message.ChannelID, r.Embed)
						if err != nil {
						}
					}
				}
			}
			r.Loading = false
			return
		}
		// Check to see if the command is ephemeral (only shown to the user)
		if r.Ephemeral {
			Session.InteractionRespond(r.Ctx.Interaction, &discordgo.InteractionResponse{
				// Ephemeral is type 64 don't ask why
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: 1 << 6,
					Embeds: []*discordgo.MessageEmbed{
						r.Embed,
					},
					Components: r.ResponseComponents.Components,
				},
			})
			return
		}

		// Default response for interaction
		err := Session.InteractionRespond(r.Ctx.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					r.Embed,
				},
				Components: r.ResponseComponents.Components,
			},
		})
		if err != nil {
			if err != nil {
				SendErrorReport(r.Ctx.Guild.ID, r.Ctx.Interaction.ChannelID, r.Ctx.Message.Author.ID, "Unable to send interaction messages", err)
			}
			if r.Ctx.Guild.Info.ResponseChannelId != "" {
				_, err = Session.ChannelMessageSendEmbed(r.Ctx.Guild.Info.ResponseChannelId, r.Embed)

			} else {
				_, err = Session.ChannelMessageSendEmbed(r.Ctx.Message.ChannelID, r.Embed)
			}

			if err != nil {
				SendErrorReport(r.Ctx.Guild.ID, r.Ctx.Interaction.ChannelID, r.Ctx.Message.Author.ID, "Unable to send message", err)
			}
		}
		return
	}
	// Try sending the response in the configured output channel
	// If that fails, try sending the response in the current channel
	// If THAT fails, send an error report
	_, err := Session.ChannelMessageSendComplex(r.Ctx.Guild.Info.ResponseChannelId, &discordgo.MessageSend{
		Embed:      r.Embed,
		Components: r.ResponseComponents.Components,
	})
	if err != nil {
		// Reply to user if no output channel
		_, err = ReplyToUser(r.Ctx.Message.ChannelID, &discordgo.MessageSend{
			Embed:      r.Embed,
			Components: r.ResponseComponents.Components,
			Reference: &discordgo.MessageReference{
				MessageID: r.Ctx.Message.ID,
				ChannelID: r.Ctx.Message.ChannelID,
				GuildID:   r.Ctx.Guild.ID,
			},
			AllowedMentions: &discordgo.MessageAllowedMentions{
				RepliedUser: false,
			},
		})
		if err != nil {
			SendErrorReport(r.Ctx.Guild.ID, r.Ctx.Message.ChannelID, r.Ctx.Message.Author.ID, "Ultimately failed to send bot response", err)
		}
	}
}

func ErrorResponse(i *discordgo.Interaction, errorMsg string, trigger string) {
	var errorEmbed = CreateEmbed(0xff3232, "Error", errorMsg, []*discordgo.MessageEmbedField{
		{
			Name:  "Command Used",
			Value: "/" + trigger,
		},
		{
			Name:  "Invoked by:",
			Value: i.Member.User.Mention(),
		},
	})
	Session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				errorEmbed,
			},
		},
	})

	time.AfterFunc(time.Second*5, func() {
		time.Sleep(time.Second * 4)
		Session.InteractionResponseDelete(Session.State.User.ID, i)
	})
}

func (r *Response) AcknowledgeInteraction() {
	Session.InteractionRespond(r.Ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "<:loadingdots:759625992166965288>",
		},
	})
	r.Loading = true
}

func ReplyToUser(channelID string, messageSend *discordgo.MessageSend) (*discordgo.Message, error) {
	return Session.ChannelMessageSendComplex(channelID, messageSend)
}
