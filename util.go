package framework

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/dlclark/regexp2"
	"regexp"
	"strconv"
	"strings"
)

// util.go
// This file contains utility functions, simplifying redundant tasks

// RemoveItem
// Remove an item from a slice by value
func RemoveItem(slice []string, delete string) []string {
	var newSlice []string
	for _, elem := range slice {
		if elem != delete {
			newSlice = append(newSlice, elem)
		}
	}
	return newSlice
}

// RemoveItems
// Removes items from a slice by index
func RemoveItems(slice []string, indexes []int) []string {
	newSlice := make([]string, len(slice))
	if len(indexes) >= len(slice) {
		return newSlice
	}
	copy(newSlice, slice)
	for _, v := range indexes {
		if len(newSlice) > v+1 && v != 0 {
			v = v - 1
		}
		//newSlice[v] = newSlice[len(newSlice)-1]
		//newSlice[len(newSlice)-1] = ""
		//newSlice = newSlice[:len(newSlice)-1]
		copy(newSlice[v:], newSlice[v+1:])    // Shift a[i+1:] left one index.
		newSlice[len(newSlice)-1] = ""        // Erase last element (write zero value).
		newSlice = newSlice[:len(newSlice)-1] // Truncate slice.
	}
	return newSlice
}

// EnsureNumbers
// Given a string, ensure it contains only numbers
// This is useful for stripping letters and formatting characters from user/role pings
func EnsureNumbers(in string) string {
	reg, err := regexp.Compile("[^0-9]+")
	if err != nil {
		log.Errorf("An unrecoverable error occurred when compiling a regex expression: %s", err)
		return ""
	}

	return reg.ReplaceAllString(in, "")
}

// EnsureLetters
// Given a string, ensure it contains only letters
// This is useful for stripping numbers from mute durations, and possibly other things
func EnsureLetters(in string) string {
	reg, err := regexp.Compile("[^a-zA-Z]+")
	if err != nil {
		log.Errorf("An unrecoverable error occurred when compiling a regex expression: %s", err)
		return ""
	}

	return reg.ReplaceAllString(in, "")
}

// CleanId
// Given a string, attempt to remove all numbers from it
// Additionally, ensure it is at least 17 characters in length
// This is a way of "cleaning" a Discord ping into a valid snowflake string
func CleanId(in string) string {
	out := EnsureNumbers(in)

	// Discord IDs must be, at minimum, 17 characters long
	if len(out) < 17 {
		return ""
	}

	return out
}

// ExtractCommand
// Given a message, attempt to extract a command trigger and command arguments out of it
// If there is no prefix, try using a bot mention as the prefix
func ExtractCommand(guild *GuildInfo, message string) (*string, *string) {
	// Check if the message starts with the bot trigger
	if strings.HasPrefix(message, guild.Prefix) {
		// Split the message on the prefix, but ensure only 2 fields are returned
		// This ensures messages containing multiple instances of the prefix don't split multiple times
		split := strings.SplitN(message, guild.Prefix, 2)

		// Get everything after the prefix as the command content
		content := split[1]

		// If the content is blank, someone used the prefix without a trigger
		if content == "" {
			return nil, nil
		}

		// Attempt to pull the trigger out of the command content by splitting on spaces
		trigger := strings.Fields(content)[0]

		// With the trigger identified, split the command content on the trigger to obtain everything BUT the trigger
		// Ensure only 2 fields are returned so it can be split further. Then, get only the second field
		fullArgs := strings.SplitN(content, trigger, 2)[1]
		fullArgs = strings.TrimPrefix(fullArgs, " ")
		// Avoids issues with strings that are case sensitive
		trigger = strings.ToLower(trigger)

		return &trigger, &fullArgs
	} else {
		// The bot can only be mentioned with a space
		botMention := Session.State.User.Mention() + " "

		// Sanitize Discord's ridiculous formatting
		message = strings.Replace(message, "!", "", 1)

		// See if someone is trying to mention the bot
		if strings.HasPrefix(message, botMention) {
			// Same process as above prefix method, but split on a bot mention instead
			split := strings.SplitN(message, botMention, 2)
			content := split[1]
			// If content is null someone just sent the prefix
			if content == "" {
				return nil, nil
			}
			trigger := strings.ToLower(strings.Fields(content)[0])
			fullArgs := strings.SplitN(content, trigger, 2)[1]
			return &trigger, &fullArgs
		} else {
			return nil, nil
		}
	}
}

// GetUser
// Given a user ID, get that user's object (global to Discord, not in a guild)
func GetUser(userId string) (*discordgo.User, error) {
	cleanedId := CleanId(userId)
	if cleanedId == "" {
		return nil, errors.New("provided ID is invalid")
	}

	return Session.User(cleanedId)
}

// logErrorReportFailure
// If an error report fails to send, log the failure
func logErrorReportFailure(recipient string, dmErr error, guildId string, channelId string, userId string, errTitle string, origErr error) {
	log.Errorf("[REPORT] Failed to DM report to %s: %s", recipient, dmErr)
	log.Error("[REPORT] ---------- BEGIN ERROR REPORT ----------")
	log.Error("[REPORT]     Report title: " + errTitle)
	// Can't .Error a nil error
	if origErr != nil {
		log.Error("[REPORT] Full error: " + origErr.Error())
	}
	log.Error("[REPORT]   Affected guild: " + guildId)
	log.Error("[REPORT] Affected channel: " + channelId)
	log.Error("[REPORT]    Affected user: " + userId)
	log.Error("[REPORT] ----------- END ERROR REPORT -----------")
}

// SendErrorReport
// Send an error report as a DM to all of the registered bot administrators
func SendErrorReport(guildId string, channelId string, userId string, title string, err error) {
	// Log a general error
	log.Errorf("[REPORT] %s (%s)", title, err)

	// Iterate through all the admins
	for admin := range botAdmins {

		// Get the channel ID of the user to DM
		dmChannel, dmCreateErr := Session.UserChannelCreate(admin)
		if dmCreateErr != nil {
			logErrorReportFailure(admin, dmCreateErr, guildId, channelId, userId, title, err)
			continue
		}

		// Create a generic embed
		reportEmbed := CreateEmbed(ColorFailure, "ERROR REPORT", title, nil)

		// Add fields if they aren't blank
		if guildId != "" {
			reportEmbed.Fields = append(reportEmbed.Fields, &discordgo.MessageEmbedField{
				Name:   "Guild ID:",
				Value:  guildId,
				Inline: false,
			})
		}

		if channelId != "" {
			reportEmbed.Fields = append(reportEmbed.Fields, &discordgo.MessageEmbedField{
				Name:   "Channel ID:",
				Value:  channelId,
				Inline: false,
			})
		}

		if userId != "" {
			reportEmbed.Fields = append(reportEmbed.Fields, &discordgo.MessageEmbedField{
				Name:   "User ID:",
				Value:  userId,
				Inline: false,
			})
		}

		if err != nil {
			reportEmbed.Fields = append(reportEmbed.Fields, &discordgo.MessageEmbedField{
				Name:   "Full error:",
				Value:  err.Error(),
				Inline: false,
			})
		}

		_, dmSendErr := Session.ChannelMessageSendEmbed(dmChannel.ID, reportEmbed)
		if dmSendErr != nil {
			logErrorReportFailure(admin, dmSendErr, guildId, channelId, userId, title, err)
			continue
		}
	}
}

// ParseTime
// Parses time strings
func ParseTime(content string) (int, string) {
	if content == "" {
		return 0, "error lol"
	}
	duration := 0

	multiplier := 1

	matches := FindAllString(TimeRegexes["all"], content)
	if len(matches) <= 0 {
		return 0, "error lol"
	}
	for _, v := range matches {
		// Grab only the letters out of the duration, to detect the unit
		muteUnit := strings.ToLower(EnsureLetters(v))

		// Grab the number out of the duration
		// Errors shouldn't be possible due to EnsureNumbers
		multiplier, _ = strconv.Atoi(EnsureNumbers(v))

		// Use the string next to the number to check how long the mute should be for
		switch muteUnit {
		case "s":
			duration = multiplier + duration
		case "m":
			duration = multiplier*60 + duration
		case "h":
			duration = multiplier*60*60 + duration
		case "d":
			duration = multiplier*60*60*24 + duration
		case "w":
			duration = multiplier*60*60*24*7 + duration
		case "y":
			duration = multiplier*60*60*24*7*52 + duration
		default:
			break
		}
	}

	return duration, createDisplayDurationString(content)
}

func createDisplayDurationString(content string) (str string) {
	// First tokenize
	str = ""
	matches := FindAllString(TimeRegexes["all"], content)
	if matches == nil || len(matches) == 0 {
		str = "Indefinite"
		return
	}
	for i, v := range matches {
		prefixChar := ""
		if i+1 == len(matches) && len(matches) > 1 {
			prefixChar = " & "
		} else if i != 0 {
			prefixChar = ", "
		}
		// Grab only the letters out of the duration, to detect the unit
		muteUnit := strings.ToLower(EnsureLetters(v))

		// Grab the number out of the duration
		// Errors shouldn't be possible due to EnsureNumbers
		multiplier, _ := strconv.Atoi(EnsureNumbers(v))

		// clean this up
		switch muteUnit {
		case "s":
			if multiplier > 1 {
				str += prefixChar + fmt.Sprintf("%d Seconds", multiplier)
				break
			}
			str += prefixChar + "Second"
			break
		case "m":
			if multiplier > 1 {
				str += prefixChar + fmt.Sprintf("%d Minutes", multiplier)
				break
			}
			str += prefixChar + fmt.Sprintf("%d Minute", multiplier)
			break
		case "h":
			if multiplier > 1 {
				str += prefixChar + fmt.Sprintf("%d Hours", multiplier)
				break
			}
			str += prefixChar + fmt.Sprintf("%d Hours", multiplier)
			break
		case "d":
			if multiplier > 1 {
				str += prefixChar + fmt.Sprintf("%d Days", multiplier)
				break
			}
			str += prefixChar + fmt.Sprintf("%d Day", multiplier)
			break
		case "w":
			if multiplier > 1 {
				str += prefixChar + fmt.Sprintf("%d Weeks", multiplier)
				break
			}
			str += prefixChar + fmt.Sprintf("%d Week", multiplier)
			break
		case "y":
			if multiplier > 1 {
				str += prefixChar + fmt.Sprintf("%d Years", multiplier)
				break
			}
			str += prefixChar + fmt.Sprintf("%d Year", multiplier)
			break
		default:
			break
		}
	}
	return
}

func FindAllString(re *regexp2.Regexp, s string) []string {
	var matches []string
	m, _ := re.FindStringMatch(s)
	for m != nil {
		matches = append(matches, m.String())
		m, _ = re.FindNextMatch(m)
	}
	return matches
}
