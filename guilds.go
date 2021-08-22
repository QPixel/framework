package framework

import (
	"errors"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// guilds.go
// This file contains the structure of a guild, and all the functions used to store and retrieve guild information

// GuildInfo
// This is all the settings and data that needs to be stored about a single guild
type GuildInfo struct {
	AddedDate               int64                  `json:"added_date"`
	ChannelDisabledCommands map[string][]string    `json:"channel_disabled_commands"`
	DeletePolicy            bool                   `json:"delete_policy"`
	GlobalDisabledCommands  []string               `json:"global_disabled_commands"`
	IgnoredChannels         []string               `json:"ignored_channels"`
	IgnoredIds              []string               `json:"ignored_ids"`
	ModeratorIds            []string               `json:"moderator_ids"`
	Prefix                  string                 `json:"prefix,"`
	ResponseChannelId       string                 `json:"response_channel_id"`
	Storage                 map[string]interface{} `json:"storage"`
	WhitelistedChannels     []string               `json:"whitelisted_channels"`
	WhitelistIds            []string               `json:"whitelist_ids"`
}

// Guild
// The definition of a guild, which is simply its ID and Info
type Guild struct {
	ID   string
	Info GuildInfo
}

// Guilds
// A map that stores the data for all known guilds
// We store pointers to the guilds, so that only one guild object is maintained across all contexts
// Otherwise, there will be information desync
var Guilds = make(map[string]*Guild)

// getGuild
// Return a Guild object corresponding to the given guildId
// If the guild doesn't exist, initialize a new guild and save it before returning
// Return a pointer to the guild object and pass that around instead, to avoid information desync
func getGuild(guildId string) *Guild {
	// The command is being ran as a dm, send back an empty guild object with default fields
	if guildId == "" {
		return &Guild{
			ID: "",
			Info: GuildInfo{
				AddedDate:               time.Now().Unix(),
				ChannelDisabledCommands: nil,
				DeletePolicy:            false,
				GlobalDisabledCommands:  nil,
				IgnoredChannels:         nil,
				IgnoredIds:              nil,
				ModeratorIds:            nil,
				Prefix:                  "!",
				ResponseChannelId:       "",
				Storage:                 make(map[string]interface{}),
				WhitelistedChannels:     nil,
				WhitelistIds:            nil,
			},
		}
	}
	if guild, ok := Guilds[guildId]; ok {
		return guild
	} else {
		// Create a new guild with default values
		newGuild := Guild{
			ID: guildId,
			Info: GuildInfo{
				AddedDate:               time.Now().Unix(),
				ChannelDisabledCommands: nil,
				DeletePolicy:            false,
				GlobalDisabledCommands:  nil,
				IgnoredChannels:         nil,
				IgnoredIds:              nil,
				ModeratorIds:            nil,
				Prefix:                  "!",
				ResponseChannelId:       "",
				Storage:                 make(map[string]interface{}),
				WhitelistedChannels:     nil,
				WhitelistIds:            nil,
			},
		}
		// Add the new guild to the map of guilds
		Guilds[guildId] = &newGuild

		// Save the guild to .json
		// A failed save is fatal, so we can count on this being successful
		newGuild.save()

		// Log that a new guild was detected
		log.Infof("New guild detected: %s", guildId)

		return &newGuild
	}
}

// GetMember
// Convenience function to get a member in this guild
// This function handles cleaning of the string so you don't have to
func (g *Guild) GetMember(userId string) (*discordgo.Member, error) {
	cleanedId := CleanId(userId)
	if cleanedId == "" {
		return nil, errors.New("invalid user ID")
	}
	return Session.GuildMember(g.ID, cleanedId)
}

// IsMember
// Determine whether or not a given userId is a member in this guild
func (g *Guild) IsMember(userId string) bool {
	_, err := g.GetMember(userId)
	if err != nil {
		return false
	}
	return true
}

// GetRole
// Convenience function to get a single role in this guild
// This function handles cleaning of the string so you don't have to
func (g *Guild) GetRole(roleId string) (*discordgo.Role, error) {
	cleanedId := CleanId(roleId)
	if cleanedId == "" {
		return nil, errors.New("invalid role ID")
	}

	roles, err := Session.GuildRoles(g.ID)

	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if role.ID == cleanedId {
			return role, nil
		}
	}

	return nil, errors.New("role not found")
}

// IsRole
// Determine whether or not a given roleId is a valid role in this guild
func (g *Guild) IsRole(roleId string) bool {
	_, err := g.GetRole(roleId)
	if err != nil {
		return false
	}
	return true
}

// HasRole
// Determine if a given user ID has a certain role in this guild
func (g *Guild) HasRole(userId string, roleId string) bool {
	member, err := g.GetMember(userId)
	if err != nil {
		return false
	}

	role, err := g.GetRole(roleId)
	if err != nil {
		return false
	}

	for _, r := range member.Roles {
		if r == role.ID {
			return true
		}
	}

	return false
}

// GetChannel
// Retrieve a single channel belonging to this guild
// This function handles cleaning of the string so you don't have to
func (g *Guild) GetChannel(channelId string) (*discordgo.Channel, error) {
	cleanedId := CleanId(channelId)
	if cleanedId == "" {
		return nil, errors.New("invalid channel ID")
	}

	channels, err := Session.GuildChannels(g.ID)
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		if channel.ID == cleanedId {
			return channel, nil
		}
	}

	return nil, errors.New("channel not found")
}

// IsChannel
// Determine whether or not a given channelId is a valid channel in this guild
func (g *Guild) IsChannel(channelId string) bool {
	_, err := g.GetChannel(channelId)
	if err != nil {
		return false
	}
	return true
}

// MemberOrRoleInList
// This is a higher-level function specifically for the Moderator, Ignored, and Whitelist checks
// Check if a given ID - member or role - exists in a given list, while automatically checking member roles if necessary
func (g *Guild) MemberOrRoleInList(checkId string, list []string) bool {
	// Check if the ID represents a member
	member, err := g.GetMember(checkId)
	if err == nil {
		// This is a member, check if their ID is found in the list directly, OR if a role they have is found in the list
		for _, id := range list {
			if member.User.ID == id {
				return true
			}
			for _, role := range member.Roles {
				if role == id {
					return true
				}
			}
		}

		// The member is not in the list, neither by ID nor by any roles they have
		return false
	}

	// Check if the ID represents a role
	role, err := g.GetRole(checkId)
	log.Infof("Role %s", role)
	if err == nil {
		// This is a role; check if this role is in the list
		for _, mod := range list {
			if role.ID == mod {
				return true
			}
		}
	}

	// All checks failed, they are not in the list
	return false
}

// SetPrefix
// Set the prefix, then save the guild data
func (g *Guild) SetPrefix(newPrefix string) {
	g.Info.Prefix = newPrefix
	g.save()
}

// IsMod
// Check if a given ID is a moderator or not
func (g *Guild) IsMod(checkId string) bool {
	return g.MemberOrRoleInList(checkId, g.Info.ModeratorIds)
}

// AddMod
// Add a user or role ID as a moderator to the bot
func (g *Guild) AddMod(addId string) error {
	// Add the ID if it is a member
	member, err := g.GetMember(addId)
	if err == nil {
		if g.IsMod(member.User.ID) {
			return errors.New("member is already a bot moderator in this guild; nothing to add")
		}
		g.Info.ModeratorIds = append(g.Info.ModeratorIds, member.User.ID)
		g.save()
		return nil
	}

	// Add the ID if it is a role
	role, err := g.GetRole(addId)
	if err == nil {
		if g.IsMod(role.ID) {
			return errors.New("role is already a bot moderator in this guild; nothing to add")
		}
		g.Info.ModeratorIds = append(g.Info.ModeratorIds, role.ID)
		g.save()
		return nil
	}

	return errors.New("failed to locate member or role")
}

// RemoveMod
// Remove a user or role ID from the list of bot moderators
func (g *Guild) RemoveMod(remId string) error {
	cleanedId := CleanId(remId)
	if cleanedId == "" {
		return errors.New("provided ID is invalid")
	}

	if !g.IsMod(cleanedId) {
		return errors.New("id is not a bot moderator in this guild; nothing to remove")
	}

	g.Info.ModeratorIds = RemoveItem(g.Info.ModeratorIds, cleanedId)
	g.save()
	return nil
}

// MemberOrRoleIsWhitelisted
// Check if a given user or role is whitelisted
// If the whitelist is empty, return true
func (g *Guild) MemberOrRoleIsWhitelisted(checkId string) bool {
	// Check if the whitelist is empty. If it is, return true immediately
	if len(g.Info.WhitelistIds) == 0 {
		return true
	}

	return g.MemberOrRoleInList(checkId, g.Info.WhitelistIds)
}

// AddMemberOrRoleToWhitelist
// Add a member OR role ID to the list of whitelisted ids
func (g *Guild) AddMemberOrRoleToWhitelist(addId string) error {
	// Make sure the id is a member or a role
	if !g.IsMember(addId) && !g.IsRole(addId) {
		return errors.New("provided ID is neither a member or a role")
	}

	cleanedId := CleanId(addId)
	if cleanedId == "" {
		return errors.New("provided ID is invalid")
	}

	if g.MemberOrRoleIsWhitelisted(cleanedId) {
		return errors.New("id is already whitelisted in this guild; nothing to add")
	}

	g.Info.WhitelistIds = append(g.Info.WhitelistIds, cleanedId)
	g.save()

	// If this ID is ignored, remove it from the ignore list, as these are mutually exclusive
	if g.MemberOrRoleIsIgnored(cleanedId) {
		err := g.RemoveMemberOrRoleFromIgnored(cleanedId)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveMemberOrRoleFromWhitelist
// Remove a given ID from the list of whitelisted IDs
func (g *Guild) RemoveMemberOrRoleFromWhitelist(remId string) error {
	cleanedId := CleanId(remId)
	if cleanedId == "" {
		return errors.New("provided ID is invalid")
	}

	if !g.MemberOrRoleIsWhitelisted(cleanedId) {
		return errors.New("id is not whitelisted in this guild; nothing to remove")
	}

	g.Info.WhitelistIds = RemoveItem(g.Info.WhitelistIds, cleanedId)
	g.save()
	return nil
}

// MemberOrRoleIsIgnored
// Determine if a given user or role ID is on the ignored list, OR if they have a role on the ignored list
// On error, treat as if they are on this list
func (g *Guild) MemberOrRoleIsIgnored(checkId string) bool {
	// Check if the ignore list is empty. If it is, return false immediately
	if len(g.Info.IgnoredIds) == 0 {
		return false
	}

	return g.MemberOrRoleInList(checkId, g.Info.IgnoredIds)
}

// AddMemberOrRoleToIgnored
// Add a user OR role ID to the list of ignored IDs
func (g *Guild) AddMemberOrRoleToIgnored(addId string) error {
	// Make sure the id is a member or a role
	if !g.IsMember(addId) && !g.IsRole(addId) {
		return errors.New("provided ID is neither a member or a role")
	}

	cleanedId := CleanId(addId)
	if cleanedId == "" {
		return errors.New("provided ID is invalid")
	}

	if g.MemberOrRoleIsIgnored(cleanedId) {
		return errors.New("id is already ignored in this guild; nothing to add")
	}

	g.Info.IgnoredIds = append(g.Info.IgnoredIds, cleanedId)
	g.save()

	// If this ID is whitelisted, remove it from the whitelist, as these are mutually exclusive
	if g.MemberOrRoleIsWhitelisted(cleanedId) {
		err := g.RemoveMemberOrRoleFromWhitelist(cleanedId)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveMemberOrRoleFromIgnored
// Remove a given ID from the list of ignored IDs
func (g *Guild) RemoveMemberOrRoleFromIgnored(remId string) error {
	cleanedId := CleanId(remId)
	if cleanedId == "" {
		return errors.New("provided ID is invalid")
	}

	if !g.MemberOrRoleIsIgnored(cleanedId) {
		return errors.New("id is not ignored in this guild; nothing to remove")
	}

	g.Info.IgnoredIds = RemoveItem(g.Info.IgnoredIds, cleanedId)
	g.save()
	return nil
}

// ChannelIsWhitelisted
// Determine if a channel ID is whitelisted. Return true if the whitelist is empty
func (g *Guild) ChannelIsWhitelisted(channelId string) bool {
	if len(g.Info.WhitelistedChannels) == 0 {
		return true
	}

	// Make sure it is a channel
	channel, err := g.GetChannel(channelId)
	if err != nil {
		return false
	}

	for _, whitelisted := range g.Info.WhitelistedChannels {
		if channel.ID == whitelisted {
			return true
		}
	}

	return false
}

// AddChannelToWhitelist
// Add a channel to the list of channels that are whitelisted (where commands can be run)
func (g *Guild) AddChannelToWhitelist(channelId string) error {
	cleanedId := CleanId(channelId)
	if cleanedId == "" {
		return errors.New("provided ID is invalid")
	}

	// Make sure it is a channel
	channel, err := g.GetChannel(cleanedId)
	if err != nil {
		return err
	}

	// Make sure it's not already in the whitelist
	if g.ChannelIsWhitelisted(channel.ID) {
		return errors.New("channel is already whitelisted")
	}

	// Add the ID to the whitelist
	g.Info.WhitelistedChannels = append(g.Info.WhitelistedChannels, channel.ID)
	g.save()

	// If this channel is ignored, remove it from the ignore list, as these are mutually exclusive
	if g.ChannelIsIgnored(channel.ID) {
		err := g.RemoveChannelFromIgnored(channel.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveChannelFromWhitelist
// Remove a channel from the list of channels that are whitelisted (where commands can be run)
func (g *Guild) RemoveChannelFromWhitelist(channelId string) error {
	cleanedId := CleanId(channelId)
	if cleanedId == "" {
		return errors.New("provided ID is invalid")
	}

	// Make check if it's even on the channel whitelist
	if !g.ChannelIsWhitelisted(cleanedId) {
		return errors.New("channel is already whitelisted")
	}

	// Remove the ID from the whitelist
	g.Info.WhitelistedChannels = RemoveItem(g.Info.WhitelistedChannels, cleanedId)
	g.save()

	return nil
}

// ChannelIsIgnored
// Determine if a channel ID is ignored. Return false if the ignore list is empty
func (g *Guild) ChannelIsIgnored(channelId string) bool {
	if len(g.Info.IgnoredChannels) == 0 {
		return false
	}

	// Make sure it is a channel
	channel, err := g.GetChannel(channelId)
	if err != nil {
		return true
	}

	for _, ignored := range g.Info.IgnoredChannels {
		if channel.ID == ignored {
			return true
		}
	}

	return false
}

// AddChannelToIgnored
// Add a channel to the list of channels that are ignored (where commands can't be run)
func (g *Guild) AddChannelToIgnored(channelId string) error {
	cleanedId := CleanId(channelId)
	if cleanedId == "" {
		return errors.New("provided ID is invalid")
	}

	// Make sure it is a channel
	channel, err := g.GetChannel(cleanedId)
	if err != nil {
		return err
	}

	// Make sure it's not already in the ignored list
	if g.ChannelIsIgnored(channel.ID) {
		return errors.New("channel is already ignored")
	}

	// Add the ID to the ignored list
	g.Info.IgnoredChannels = append(g.Info.IgnoredChannels, channel.ID)
	g.save()

	// If this channel is whitelisted, remove it from the whitelist, as these are mutually exclusive
	if g.ChannelIsWhitelisted(channel.ID) {
		err := g.RemoveChannelFromWhitelist(channel.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveChannelFromIgnored
// Remove a channel from the list of channels that are ignored (where commands can't be run)
func (g *Guild) RemoveChannelFromIgnored(channelId string) error {
	cleanedId := CleanId(channelId)
	if cleanedId == "" {
		return errors.New("provided ID is invalid")
	}

	// Make check if it's even on the ignored channel list
	if !g.ChannelIsIgnored(cleanedId) {
		return errors.New("channel is not ignored")
	}

	// Remove the ID from the ignore list
	g.Info.IgnoredChannels = RemoveItem(g.Info.IgnoredChannels, cleanedId)
	g.save()

	return nil
}

// IsGloballyDisabled
// Check if a given command is globally disabled
func (g *Guild) IsGloballyDisabled(trigger string) bool {
	for _, disabled := range g.Info.GlobalDisabledCommands {
		if strings.ToLower(disabled) == strings.ToLower(trigger) {
			return true
		}
	}

	return false
}

// EnableCommandGlobally
// Remove a command from the list of *globally disabled* triggers
func (g *Guild) EnableCommandGlobally(trigger string) error {
	if !g.IsGloballyDisabled(trigger) {
		return errors.New("trigger is not disabled; nothing to enable")
	}

	g.Info.GlobalDisabledCommands = RemoveItem(g.Info.GlobalDisabledCommands, trigger)
	g.save()
	return nil
}

// DisableCommandGlobally
// Add a command to the list of *globally disabled* commands
func (g *Guild) DisableCommandGlobally(command string) error {
	if g.IsGloballyDisabled(command) {
		return errors.New("command is not enabled; nothing to disable")
	}

	g.Info.GlobalDisabledCommands = append(g.Info.GlobalDisabledCommands, command)
	g.save()
	return nil
}

// CommandIsDisabledInChannel
// Check if a given command is disabled in the given channel
func (g *Guild) CommandIsDisabledInChannel(command string, channelId string) bool {
	cleanedId := CleanId(channelId)
	if cleanedId == "" {
		return true
	}

	if !g.IsChannel(cleanedId) {
		return true
	}

	// Iterate over every channel ID (the map key) and their internal list of disabled triggers
	for channel, commands := range g.Info.ChannelDisabledCommands {

		// If the channel matches our current channel, continue
		if channel == cleanedId {

			// For every disabled trigger in the list...
			for _, disabled := range commands {

				// If the current trigger matches a disabled one, return true
				if disabled == command {
					return true
				}
			}
		}
	}

	return false
}

// EnableCommandInChannel
// Given a command and channel ID, remove that command from that channel's list of blocked comamnds
func (g *Guild) EnableCommandInChannel(command string, channelId string) error {
	cleanedId := CleanId(channelId)
	if cleanedId == "" {
		return errors.New("provided channel ID is invalid")
	}

	if !g.CommandIsDisabledInChannel(command, cleanedId) {
		return errors.New("that command is not disabled in this channel; nothing to enable")
	}

	// Remove the trigger from THIS channel's list
	g.Info.ChannelDisabledCommands[cleanedId] = RemoveItem(g.Info.ChannelDisabledCommands[cleanedId], command)

	// If there are no more items, delete the entire channel list, otherwise it will appear as null in the json
	if len(g.Info.ChannelDisabledCommands[cleanedId]) == 0 {
		delete(g.Info.ChannelDisabledCommands, cleanedId)
	}

	g.save()
	return nil
}

// DisableCommandInChannel
// Given a command and channel ID, add that command to that channel's list of blocked commands
func (g *Guild) DisableTriggerInChannel(command string, channelId string) error {
	cleanedId := CleanId(channelId)
	if cleanedId == "" {
		return errors.New("provided channel ID is invalid")
	}

	if g.CommandIsDisabledInChannel(command, cleanedId) {
		return errors.New("that trigger is already disabled in this channel; nothing to disable")
	}

	g.Info.ChannelDisabledCommands[cleanedId] = append(g.Info.ChannelDisabledCommands[cleanedId], command)
	g.save()
	return nil
}

// SetDeletePolicy
// Set the delete policy, then save the guild data
func (g *Guild) SetDeletePolicy(policy bool) {
	g.Info.DeletePolicy = policy
	g.save()
}

// SetResponseChannel
// Check that the channel exists, set the response channel, then save the guild data
func (g *Guild) SetResponseChannel(channelId string) error {
	// If channelId is blank,
	if channelId == "" {
		g.Info.ResponseChannelId = channelId
		g.save()
		return nil
	}
	// Try grabbing the channel first (we don't use IsChannel since we need the real ID)
	channel, err := g.GetChannel(channelId)
	if err != nil {
		return err
	}
	g.Info.ResponseChannelId = channel.ID
	g.save()
	return nil
}

// Kick
// Kick a member
func (g *Guild) Kick(userId string, reason string) error {
	// Make sure the member exists
	member, err := g.GetMember(userId)
	if err != nil {
		return err
	}

	// Kick the member
	if reason != "" {
		return Session.GuildMemberDeleteWithReason(g.ID, member.User.ID, reason)
	} else {
		return Session.GuildMemberDelete(g.ID, member.User.ID)
	}
}

// Ban
// Ban a user, who may not be a member
func (g *Guild) Ban(userId string, reason string, deleteDays int) error {
	// Make sure the USER exists, because they may not be a member
	user, err := GetUser(userId)
	if err != nil {
		return err
	}

	// Ban the member
	if reason != "" {
		return Session.GuildBanCreateWithReason(g.ID, user.ID, reason, deleteDays)
	} else {
		return Session.GuildBanCreate(g.ID, user.ID, deleteDays)
	}
}

// PurgeChannel
// Purge the last N messages in a given channel, regardless of user
func (g *Guild) PurgeChannel(channelId string, deleteCount int) (int, error) {
	// Make sure the channel exists
	channel, err := g.GetChannel(channelId)
	if err != nil {
		return 0, err
	}

	// Get the group of messages to delete
	deleteGroup, err := Session.ChannelMessages(channel.ID, deleteCount, "", "", "")
	if err != nil {
		return 0, err
	}

	// Convert the messages to IDs
	// For some reason, discordgo has decided to not allow message objects in the delete function...
	var messageIds []string
	for _, message := range deleteGroup {
		messageIds = append(messageIds, message.ID)
	}

	// Delete the messages
	return len(messageIds), Session.ChannelMessagesBulkDelete(channel.ID, messageIds)
}

// PurgeUserInChannel
// Purge a user's messages in a certain channel
// Delete deleteCount messages, searching through a maximum of searchCount messages
func (g *Guild) PurgeUserInChannel(userId string, channelId string, deleteCount int) (int, error) {
	// Make sure the channel exists
	channel, err := g.GetChannel(channelId)
	if err != nil {
		return 0, err
	}

	// Make sure the user exists
	deleteUser, err := GetUser(userId)
	if err != nil {
		return 0, err
	}

	// Start compiling the messages to delete, in batches of 100
	var deleteIds []string
	lastId := ""

	// Search a maximum of 300 messages, loop 3 times
	for i := 0; i < 3; i++ {
		// Break out of the loop if we've got the amount of messages we needed
		if deleteCount <= len(deleteIds) {
			break
		}

		// Get 100 messages from the channel in this iteration
		deleteGroup, err := Session.ChannelMessages(channel.ID, 100, lastId, "", "")
		if err != nil {
			// If we don't have any IDs to delete yet, return an error
			// Break early otherwise
			if len(deleteIds) == 0 {
				return 0, err
			} else {
				break
			}
		}

		// If no messages were returned, break
		if len(deleteGroup) == 0 {
			break
		}

		// Set the last ID so we can keep searching up for messages before this
		lastId = deleteGroup[len(deleteGroup)-1].ID

		// Go through all the returned messages, and search for messages written by the author we're looking for
		for _, message := range deleteGroup {
			if deleteCount <= len(deleteIds) {
				break
			}
			if message.Author.ID == deleteUser.ID {
				deleteIds = append(deleteIds, message.ID)
			}
		}
	}

	// If we got messages to delete, delete them
	if len(deleteIds) != 0 {
		return len(deleteIds), Session.ChannelMessagesBulkDelete(channel.ID, deleteIds)
	} else {
		return 0, nil
	}

}

// PurgeUser
// PurgeUser a user's messages in any channel
func (g *Guild) PurgeUser(userId string, deleteCount int) (int, error) {
	// Get all the channels in the guild
	channels, err := Session.GuildChannels(g.ID)
	if err != nil {
		return 0, err
	}

	// Systematically check all channels in the guild for messages to delete
	totalDeleted := 0
	for _, channel := range channels {
		// Break if we've deleted the amount we wanted to delete
		if deleteCount <= totalDeleted {
			break
		}

		// Don't bother checking user ID, because this function will do it automatically, reducing API calls
		numDeleted, err := g.PurgeUserInChannel(userId, channel.ID, deleteCount-totalDeleted)
		if err != nil {
			return 0, err
		}
		totalDeleted += numDeleted
	}

	return totalDeleted, nil
}

// StoreString
// Store a string to this guild's arbitrary storage
func (g *Guild) StoreString(key string, value string) {
	g.Info.Storage[key] = value
	g.save()
}

// GetString
// Retrieve a string from this guild's arbitrary storage, and error if the cast fails
func (g *Guild) GetString(key string) (string, error) {
	res, ok := g.Info.Storage[key].(string)
	if !ok {
		return "", errors.New("failed to cast the data to type \"string\"")
	}

	return res, nil
}

// StoreInt64
// Store an int64 to this guild's arbitrary storage
func (g *Guild) StoreInt64(key string, value int64) {
	g.Info.Storage[key] = value
	g.save()
}

// GetInt64
// Retrieve an int64 from this guild's arbitrary storage, and error if the cast fails
func (g *Guild) GetInt64(key string) (int64, error) {
	res, ok := g.Info.Storage[key].(int64)
	if !ok {
		return -1, errors.New("failed to cast the data to type \"int64\"")
	}

	return res, nil
}

// StoreMap
// Store a map to this guild's arbitrary storage
func (g *Guild) StoreMap(key string, value map[string]interface{}) {
	g.Info.Storage[key] = value
	g.save()
}

// GetMap
// Get a map from this guild's arbitrary storage, and error if the cast fails
func (g *Guild) GetMap(key string) (map[string]interface{}, error) {
	res, ok := g.Info.Storage[key].(map[string]interface{})
	if !ok {
		return nil, errors.New("failed to cast the data to type \"map[string]interface{}\"")
	}

	return res, nil
}

// GetCommandUsage
//// Compile the usage information for a single command, so it can be printed out
func (g *Guild) GetCommandUsage(cmd CommandInfo) string {
	// Get the trigger for the command, and add the prefix to it
	trigger := g.Info.Prefix + cmd.Trigger

	// If there are no usage examples, we only need to print the trigger, wrapped in code formatting
	if len(cmd.Arguments.Keys()) == 0 {
		return "```\n" + trigger + "\n```"
	}

	// Start building the output
	output := "\n\n"
	cnt := 0

	for _, arg := range cmd.Arguments.Keys() {
		v, ok := cmd.Arguments.Get(arg)
		if !ok {
			return "```\n" + trigger + "\n```"
		}
		argType := v.(*ArgInfo)
		output += trigger + " <" + arg + "> (" + argType.Description + ") "
		if cnt != len(cmd.Arguments.Keys())-1 {
			output += "\n"
		}
		cnt++
	}
	return "```\n" + output + "\n```"
}
