package framework

import (
	"reflect"
	"time"

	"github.com/bwmarrin/discordgo"
)

// response.go
// This file contains structures and functions that make it easier to create and send response embeds

// ResponseComponents
// Stores the components for response
// allows for functions to add data
type ResponseComponents struct {
	Components        []discordgo.ActionsRow
	SelectMenuOptions []discordgo.SelectMenuOption
}

// Response
// The Response type, can be build and sent to a given guild
type Response struct {
	Ctx                *Context
	Success            bool
	Loading            bool
	Ephemeral          bool
	Reply              bool
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

func (c *ResponseComponents) FindButton(customID string) (*discordgo.Button, bool) {
	log.Debugf("%#v", c.Components)
	for _, row := range c.Components {
		for _, component := range row.Components {
			log.Debugf("%#v", component)
			if component.Type() == discordgo.ButtonComponent {
				ctype := reflect.TypeOf(component).Kind()
				if ctype == reflect.Ptr {
					if component.(*discordgo.Button).CustomID == customID {
						return component.(*discordgo.Button), true
					}
				} else {
					if component.(discordgo.Button).CustomID == customID {
						return component.(*discordgo.Button), true
					}
				}

			}
		}
	}
	return nil, false
}

func (c *ResponseComponents) FindDropDown(customID string) (*discordgo.SelectMenu, bool) {
	for _, row := range c.Components {
		for _, component := range row.Components {
			if component.(discordgo.SelectMenu).CustomID == customID {
				return component.(*discordgo.SelectMenu), true
			}
		}
	}
	return nil, false
}

func (c *ResponseComponents) SetButton(customID string, button discordgo.Button, row ...int) {
	if len(row) == 0 {
		row = append(row, 0)
	}
	c.Components[row[0]].Components = append(c.Components[row[0]].Components, &button)
}

func (c *ResponseComponents) ReplaceButton(customID string, button discordgo.Button) {
	for i, row := range c.Components {
		for j, component := range row.Components {
			ctype := reflect.TypeOf(component).Kind()
			if ctype == reflect.Ptr {
				if component.(*discordgo.Button).CustomID == customID {
					c.Components[i].Components[j] = &button
				}
			} else {
				if component.(discordgo.Button).CustomID == customID {
					c.Components[i].Components[j] = &button
				}
			}
		}
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
		Reply:     ephemeral,
	}
	if messageComponents {
		r.ResponseComponents.Components = MakeActionRow()
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

	return r
}

// ReconstructResponse
// Reconstruct a response object from a given context. Only for interactions
func ReconstructResponse(ctx *Context) *Response {
	if ctx.Interaction == nil {
		log.Errorf("Tried to reconstruct response from context without interaction")
		return nil
	}
	if ctx.Interaction.Message == nil {
		log.Errorf("Tried to reconstruct response from context without interaction message")
		return nil
	}
	if len(ctx.Interaction.Message.Embeds) == 0 {
		log.Errorf("Tried to reconstruct response from context without embeds")
		return nil
	}
	log.Debugf("Reconstructing response from context %#v", ctx.Interaction)
	r := &Response{
		Ctx:   ctx,
		Embed: ctx.Interaction.Message.Embeds[0],
		ResponseComponents: &ResponseComponents{
			Components: ConvertMessageComponent(ctx.Interaction.Message.Components),
		},
		Loading:   ctx.Cmd.IsTyping,
		Ephemeral: ctx.Interaction.Message.Flags == 1<<6,
		Reply:     false,
	}
	return r
}

// ConvertComponent
// Properly Type Asserts a MessageComponent to any of the possible types, and returns it
// func ConvertComponent[K discordgo.MessageComponent](component discordgo.MessageComponent) (K, bool) {}

// ConvertMessageComponent
// Converts the components on the message struct to an array of ActionsRow
func ConvertMessageComponent(components []discordgo.MessageComponent) []discordgo.ActionsRow {
	var rows []discordgo.ActionsRow
	log.Debugf("Converting message components: %#v", components)
	for _, component := range components {
		if row, ok := component.(discordgo.ActionsRow); ok {
			rows = append(rows, row)
		} else if row, ok := component.(*discordgo.ActionsRow); ok {
			rows = append(rows, *row)
		}
	}
	return rows
}

// MakeActionRow
// Returns a slice of a Message Component, containing a singular ActionsRow
func MakeActionRow() []discordgo.ActionsRow {
	return make([]discordgo.ActionsRow, 1)
}

// SerializeActionRow
// Converts a slice of ActionsRow to a slice of Message Components
func SerializeActionRow(row []discordgo.ActionsRow) *[]discordgo.MessageComponent {
	var components []discordgo.MessageComponent
	for _, r := range row {
		components = append(components, r)
	}
	return &components
}

// ConvertToMessageComponent
// Converts a slice of Message Components to a slice of Message Components
func ConvertToMessageComponent[T []discordgo.MessageComponent](component T) *[]discordgo.MessageComponent {
	if c, ok := (any(component).([]discordgo.MessageComponent)); ok {
		return &c
	}
	return nil
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
		Emoji:    nil,
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
func (r *Response) AppendButton(label string, style discordgo.ButtonStyle, url string, customID string, rowID ...int) {
	if len(rowID) == 0 {
		rowID = append(rowID, 0)
	}
	if r.ResponseComponents.Components == nil {
		r.ResponseComponents.Components = MakeActionRow()
	}
	button := CreateButton(label, style, customID, url, false)
	r.ResponseComponents.SetButton(customID, *button, rowID...)
}

// AppendDropDown
// Adds a DropDown component
func (r *Response) AppendDropDown(customID string, placeholder string, noNewRow bool) {
	if noNewRow {
		row := r.ResponseComponents.Components[0]
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
				Components: *SerializeActionRow(r.ResponseComponents.Components),
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
				components := SerializeActionRow(r.ResponseComponents.Components)
				log.Debugf("Sending interaction response with components: %#v", components)
				_, err := Session.InteractionResponseEdit(r.Ctx.Interaction, &discordgo.WebhookEdit{
					Components: components,
					Embeds: &[]*discordgo.MessageEmbed{
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
				components := SerializeActionRow(r.ResponseComponents.Components)
				log.Debugf("Sending interaction response with components: %#v", components)
				_, err := Session.InteractionResponseEdit(r.Ctx.Interaction, &discordgo.WebhookEdit{
					Content: ToPtr[string](""),
					Embeds: &[]*discordgo.MessageEmbed{
						r.Embed,
					},
					Components: components,
				})
				// Just in case the interaction gets removed.
				if err != nil {
					log.Errorf("Error sending interaction response: %s", err)
					_, err := Session.ChannelMessageSendEmbed(r.Ctx.Guild.Info.ResponseChannelId, r.Embed)
					if err != nil {
						_, _ = Session.ChannelMessageSendEmbed(r.Ctx.Message.ChannelID, r.Embed)
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
					Components: *SerializeActionRow(r.ResponseComponents.Components),
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
				Components: *SerializeActionRow(r.ResponseComponents.Components),
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
		Components: *SerializeActionRow(r.ResponseComponents.Components),
	})
	if err != nil && r.Reply {
		// Reply to user if no output channel
		_, err = ReplyToUser(r.Ctx.Message.ChannelID, &discordgo.MessageSend{
			Embed:      r.Embed,
			Components: *SerializeActionRow(r.ResponseComponents.Components),
			Reference: &discordgo.MessageReference{
				MessageID: r.Ctx.Message.ID,
				ChannelID: r.Ctx.Message.ChannelID,
				GuildID:   r.Ctx.Guild.ID,
			},
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{},
			},
		})
		if err != nil {
			SendErrorReport(r.Ctx.Guild.ID, r.Ctx.Message.ChannelID, r.Ctx.Message.Author.ID, "Ultimately failed to send bot response", err)
		}
	} else if !r.Reply {
		// If the command does not want to reply lets just send it to the channel the command was invoked
		_, err = Session.ChannelMessageSendComplex(r.Ctx.Message.ChannelID, &discordgo.MessageSend{
			Embed:      r.Embed,
			Components: *SerializeActionRow(r.ResponseComponents.Components),
		})
	}
}

// -- Response Editing --

// EditButtonDisabled
// Edit a button to be disabled
func (r *Response) EditButtonDisabled(buttonID string) {
	r.EditButtonComplex(buttonID, "", 0, "", true)
}

// EditButtonComplex
// Edit a button
func (r *Response) EditButtonComplex(buttonID string, label string, style discordgo.ButtonStyle, url string, disabled bool) {
	button, ok := r.ResponseComponents.FindButton(buttonID)
	if !ok {
		log.Errorf("Could not find button with ID %s", buttonID)
		return
	}

	if label != "" {
		button.Label = label
	}
	if style != 0 {
		button.Style = style
	}
	if url != "" {
		button.URL = url
	}

	if disabled {
		button.Disabled = true
	}
	r.ResponseComponents.ReplaceButton(buttonID, *button)
}

// Edit
// Edit a response
func (r *Response) Edit() {
	component := SerializeActionRow(r.ResponseComponents.Components)
	log.Debugf("Editing response with components: %#v", component)
	_, err := Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    r.Ctx.Interaction.Message.ChannelID,
		ID:         r.Ctx.Interaction.Message.ID,
		Embed:      r.Embed,
		Components: component,
	})
	if err != nil {
		SendErrorReport(r.Ctx.Guild.ID, r.Ctx.Message.ChannelID, r.Ctx.Message.Author.ID, "Failed to edit message", err)
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
		Session.InteractionResponseDelete(i)
	})
}

func (r *Response) AcknowledgeInteraction() {
	Session.InteractionRespond(r.Ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
	})
	r.Loading = true
}

func ReplyToUser(channelID string, messageSend *discordgo.MessageSend) (*discordgo.Message, error) {
	return Session.ChannelMessageSendComplex(channelID, messageSend)
}
