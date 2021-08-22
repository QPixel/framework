package framework

import (
	"errors"
	"fmt"
	"github.com/QPixel/orderedmap"
	"github.com/bwmarrin/discordgo"
	"github.com/dlclark/regexp2"
	"strconv"
	"strings"
)

// Arguments.go
// File for all argument based functions which includes: parsing, creating, and more
// Woo

// pixel wrote this

// -- TypeDefs --

// ArgTypes
// A way to get type safety in AddArg
type ArgTypes string

var (
	ArgOption  ArgTypes = "option"
	ArgContent ArgTypes = "content"
	ArgFlag    ArgTypes = "flag"
)

// ArgTypeGuards
// A way to get type safety in AddArg
type ArgTypeGuards string

var (
	Int       ArgTypeGuards = "int"
	String    ArgTypeGuards = "string"
	Channel   ArgTypeGuards = "channel"
	User      ArgTypeGuards = "user"
	Role      ArgTypeGuards = "role"
	GuildArg  ArgTypeGuards = "guild"
	Message   ArgTypeGuards = "message"
	Boolean   ArgTypeGuards = "bool"
	Id        ArgTypeGuards = "id"
	SubCmd    ArgTypeGuards = "subcmd"
	SubCmdGrp ArgTypeGuards = "subcmdgrp"
	ArrString ArgTypeGuards = "arrString"
	Time      ArgTypeGuards = "time"
)

// ArgInfo
// Describes a CommandInfo argument
type ArgInfo struct {
	Match         ArgTypes
	TypeGuard     ArgTypeGuards
	Description   string
	Required      bool
	Flag          bool
	DefaultOption string
	Choices       []string
	Regex         *regexp2.Regexp
}

// CommandArg
// Describes what a cmd ctx will receive
type CommandArg struct {
	info  ArgInfo
	Value interface{}
}

// Arguments
// Type of the arguments field in the command ctx
type Arguments map[string]CommandArg

// -- Command Configuration --

// CreateCommandInfo
// Creates a pointer to a CommandInfo
func CreateCommandInfo(trigger string, description string, public bool, group Group) *CommandInfo {
	cI := &CommandInfo{
		Aliases:     nil,
		Arguments:   orderedmap.New(),
		Description: description,
		Group:       group,
		Public:      public,
		IsTyping:    false,
		Trigger:     trigger,
	}
	return cI
}

// CreateRawCmdInfo
// Creates a pointer to a CommandInfo
func CreateRawCmdInfo(cI *CommandInfo) *CommandInfo {
	cI.Arguments = orderedmap.New()
	return cI
}

// SetParent
// Sets the parent properties
func (cI *CommandInfo) SetParent(isParent bool, parentID string) {
	if !isParent {
		cI.IsChild = true
	}
	cI.IsParent = isParent
	cI.ParentID = parentID
}

//AddCmdAlias
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
func (cI *CommandInfo) AddArg(argument string, typeGuard ArgTypeGuards, match ArgTypes, description string, required bool, defaultOption string) *CommandInfo {
	cI.Arguments.Set(argument, &ArgInfo{
		TypeGuard:     typeGuard,
		Description:   description,
		Required:      required,
		Match:         match,
		DefaultOption: defaultOption,
		Choices:       nil,
		Regex:         nil,
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
		log.Fatalf("Unable to create regex for flag on command %s flag: %s", cI.Trigger, flag)
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

// AddChoices
// Adds SubCmd choices
func (cI *CommandInfo) AddChoices(arg string, choices []string) *CommandInfo {
	v, ok := cI.Arguments.Get(arg)
	if ok {
		vv := v.(*ArgInfo)
		vv.Choices = choices
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

//todo subcommand stuff
//// BindToChoice
//// Bind an arg to choice (subcmd)
//func (cI *CommandInfo) BindToChoice(arg string, choice string) {
//
//}

// CreateAppOptSt
// Creates an ApplicationOptionsStruct for all the args.
func (cI *CommandInfo) CreateAppOptSt() *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{}
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

/* Argument Parsing Helpers */

func createContentString(splitString []string, currentPos int) (string, int) {
	str := ""
	for i := currentPos; i < len(splitString); i++ {
		str += splitString[i] + " "
		currentPos = i
	}
	return strings.TrimSuffix(str, " "), currentPos
}

// Finds all the 'option' type args
func findAllOptionArgs(argString []string, keys []string, infoArgs *orderedmap.OrderedMap, args *Arguments) (Arguments, bool, []string, []string) {
	if len(keys) == 0 || keys == nil {
		return *args, false, []string{}, []string{}
	}
	modifiedArgString := ""
	var modKeys []string
	var indexes []int

	// (semi) Brute force method
	// First lets find all required args
	currentPos := 0
	for i, v := range keys {
		// error handling
		iA, ok := infoArgs.Get(v)
		if !ok {
			err := errors.New(fmt.Sprintf("Unable to find map relating to key: %s", keys[i]))
			SendErrorReport("", "", "", "Argument Parsing error", err)
			continue
		}
		vv := iA.(*ArgInfo)
		if vv.Match == ArgContent {
			return *args, true, argString, keys
		}
		if vv.Required {
			if vv.TypeGuard != String {
				var value string
				value, argString = findTypeGuard(strings.Join(argString, " "), argString, vv.TypeGuard)
				(*args)[v] = handleArgOption(value, *vv)
				indexes = append(indexes, i)
			} else if checkTypeGuard(argString[currentPos], vv.TypeGuard) {
				(*args)[v] = handleArgOption(argString[currentPos], *vv)
				currentPos++
				indexes = append(indexes, i)
			} else {
				(*args)[v] = handleArgOption(vv.DefaultOption, *vv)
				indexes = append(indexes, i)
				continue
			}
		} else {
			break
		}
	}
	// Remove already found keys and clear the index list
	// We also reset some values that we reuse
	//if
	modKeys = RemoveItems(keys, indexes)
	argString = argString[currentPos:]
	indexes = nil
	currentPos = 0
	// Return early if the argument parser has found all args
	if argString == nil || len(argString) == 0 || len(modKeys) == 0 || modKeys == nil {
		return *args, false, argString, modKeys
	}

	// Now lets find the not required args
	for i, v := range modKeys {
		// error handling
		iA, ok := infoArgs.Get(v)
		if !ok {
			err := errors.New(fmt.Sprintf("Unable to find map relating to key: %s", v))
			SendErrorReport("", "", "", "Argument Parsing error", err)
			continue
		}
		vv := iA.(*ArgInfo)
		// If we find an arg that is required send an error and return
		if vv.Required {
			err := errors.New(fmt.Sprintf("Found a required arg where there is supposed to be none %s", v))
			SendErrorReport("", "", "", "Argument Parsing error", err)
			break
		}
		if vv.Match == ArgContent {
			modKeys = RemoveItems(modKeys, indexes)
			return *args, true, argString, modKeys
		}
		// Break early if current pos is the length of the array
		if currentPos == len(argString) {
			break
		}
		if vv.TypeGuard != String {
			var value string
			value, argString = findTypeGuard(strings.Join(argString, " "), argString, vv.TypeGuard)
			(*args)[v] = handleArgOption(value, *vv)
			indexes = append(indexes, i)
		} else if checkTypeGuard(argString[currentPos], vv.TypeGuard) {
			(*args)[v] = handleArgOption(argString[currentPos], *vv)
			currentPos++
			indexes = append(indexes, i)
		} else {

		}
	}
	//
	return *args, false, createSplitString(modifiedArgString), modKeys
}

func findTypeGuard(input string, array []string, typeguard ArgTypeGuards) (string, []string) {
	switch typeguard {
	case Int:
		if match, isMatch := TypeGuard["int"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		}
		return "", array
	case Boolean:
		if match, isMatch := TypeGuard["boolean"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		}
		return "", array
	case Channel:
		if match, isMatch := MentionStringRegexes["channel"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		} else if match, isMatch := MentionStringRegexes["id"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		}
		return "", array
	case Role:
		if match, isMatch := MentionStringRegexes["role"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		} else if match, isMatch := MentionStringRegexes["id"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		}
		return "", array
	case User:
		if match, isMatch := MentionStringRegexes["user"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		} else if match, isMatch := MentionStringRegexes["id"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		}
		return "", array
	case ArrString:
		if match, isMatch := TypeGuard["arrString"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		}
		return "", array
	case Message:
		if match, isMatch := TypeGuard["message_url"].FindStringMatch(input); isMatch == nil && match != nil {
			return match.String(), RemoveItem(array, match.String())
		}
		return "", array
	case Time:
		match := strings.Join(FindAllString(TimeRegexes["all"], input), "")
		//if match, isMatch := TimeRegexes["all"].Mat(input); isMatch == nil && match != nil {
		//	return match.String(), RemoveItem(array, match.String())
		//}
		if match != "" {
			return match, RemoveItem(array, match)
		}
		return "", array
	default:
		return "", array
	}
}

func findAllFlags(argString string, keys []string, infoArgs *orderedmap.OrderedMap, args *Arguments) ([]string, Arguments, []string) {
	modifiedArgString := argString
	var indexes []int
	var modKeys []string
	for index, a := range keys {
		v, _ := infoArgs.Get(a)
		vv := v.(*ArgInfo)
		// Skip because the argument has no flag
		if !vv.Flag {
			continue
		}
		// Use the compiled regex to search the arg string for a matching result.
		match, err := vv.Regex.FindStringMatch(argString)
		// Error handling/no match
		if err != nil || match == nil {
			if vv.Match == ArgOption {
				(*args)[a] = handleArgOption(vv.DefaultOption, *vv)
			} else {
				(*args)[a] = CommandArg{info: *vv, Value: "false"}
			}
			// Set the modified arg string to the mod string
			indexes = append(indexes, index)
			continue
		}

		// Check to see if the flag is a string 'option' or a boolean 'flag'
		if vv.Match == ArgOption {
			val := strings.Trim(strings.SplitN(match.String(), " ", 2)[1], "\"")
			if checkTypeGuard(val, vv.TypeGuard) {
				(*args)[a] = handleArgOption(val, *vv)
			}
		} else if vv.Match == ArgFlag {
			(*args)[a] = CommandArg{info: *vv, Value: "true"}
		} // todo figure out if indexes need to put an else statement here

		// Replace all reference to the flag in the string.
		modString, err := vv.Regex.Replace(modifiedArgString, "", -1, -1)
		if err != nil {
			continue
		}
		// Set the modified arg string to the mod string
		modifiedArgString = modString
		indexes = append(indexes, index)
	}
	if len(indexes) > 0 {
		// set keys to nil if flags have already gotten all the args
		if len(indexes) == len(keys) {
			modKeys = nil
			return []string{}, *args, keys
		}
		modKeys = RemoveItems(keys, indexes)
	}
	if modifiedArgString == "" {
		modifiedArgString = argString
	}
	if len(modKeys) == 0 || modKeys == nil {
		modKeys = keys
	}
	return createSplitString(modifiedArgString), *args, modKeys
}

// Creates a "split" string (array of strings that is split off of spaces
func createSplitString(argString string) []string {
	splitStr := strings.SplitAfter(argString, " ")
	var newSplitStr []string
	quotedStringBuffer := ""
	isQuotedString := false
	for _, v := range splitStr {
		if v == "" || v == " " {
			continue
		}
		// Checks to see if the string is a quoted argument.
		// If so, it will combine it into one string
		if strings.Contains(v, "\"") || isQuotedString {
			if strings.HasSuffix(strings.Trim(v, " "), "\"") {
				// Trim quotes and trim space suffix
				quotedStringBuffer = strings.TrimSuffix(strings.Trim(quotedStringBuffer+strings.Trim(v, " "), "\""), " ")
				newSplitStr = append(newSplitStr, quotedStringBuffer)

				isQuotedString = false
				quotedStringBuffer = ""
				continue
			}
			isQuotedString = true
			quotedStringBuffer = quotedStringBuffer + v
			continue
		} else {
			// If the string suffix contains a whitespace character, we need to remove that
			v = strings.TrimSuffix(v, " ")
			newSplitStr = append(newSplitStr, v)
		}
	}
	return newSplitStr
}

func handleArgOption(str string, info ArgInfo) CommandArg {
	return CommandArg{
		info:  info,
		Value: str,
	}
}

func checkTypeGuard(str string, typeguard ArgTypeGuards) bool {
	switch typeguard {
	case String:
		return true
	case Int:
		if _, err := strconv.Atoi(str); err == nil {
			return true
		}
		return false
	case Boolean:
		if _, err := strconv.ParseBool(str); err == nil {
			return true
		}
	case Channel:
		if isMatch, _ := MentionStringRegexes["channel"].MatchString(str); isMatch {
			return true
		} else if isMatch, _ := MentionStringRegexes["id"].MatchString(str); isMatch {
			return true
		}
	case Role:
		if isMatch, _ := MentionStringRegexes["role"].MatchString(str); isMatch {
			return true
		} else if isMatch, _ := MentionStringRegexes["id"].MatchString(str); isMatch {
			return true
		}
	case User:
		if isMatch, _ := MentionStringRegexes["user"].MatchString(str); isMatch {
			return true
		} else if isMatch, _ := MentionStringRegexes["id"].MatchString(str); isMatch {
			return true
		}
		return false
	case ArrString:
		if isMatch, _ := TypeGuard["arrString"].MatchString(str); isMatch {
			return true
		}
		return false
	case Message:
		if isMatch, _ := TypeGuard["message_url"].MatchString(str); isMatch {
			return true
		}
		return false
	}
	return false
}

/* Argument Casting s*/

// StringValue
// Returns the string value of the arg
func (ag CommandArg) StringValue() string {
	if ag.Value == nil {
		return ""
	}
	if v, ok := ag.Value.(string); ok {
		return v
	} else if v := strconv.FormatFloat(ag.Value.(float64), 'f', 2, 64); v != "" {
		return v
	} else if v = strconv.FormatBool(ag.Value.(bool)); v != "" {
		return v
	}
	return ""
}

// Int64Value
// Returns the int64 value of the arg
func (ag CommandArg) Int64Value() int64 {
	if ag.Value == nil {
		return 0
	}
	if v, ok := ag.Value.(float64); ok {
		return int64(v)
	} else if v, err := strconv.ParseInt(ag.StringValue(), 10, 64); err == nil {
		return v
	}
	return 0
}

// IntValue
// Returns the int value of the arg
func (ag CommandArg) IntValue() int {
	if ag.Value == nil {
		return 0
	}
	if v, ok := ag.Value.(float64); ok {
		return int(v)
	} else if v, err := strconv.Atoi(ag.StringValue()); err == nil {
		return v
	}
	return 0
}

// FloatValue
// Returns the int value of the arg
func (ag CommandArg) FloatValue() float64 {
	if ag.Value == nil {
		return 0.0
	}
	if v, ok := ag.Value.(float64); ok {
		return v
	} else if v, err := strconv.ParseFloat(ag.StringValue(), 64); err == nil {
		return v
	}
	return 0.0
}

// BoolValue
// Returns the int value of the arg
func (ag CommandArg) BoolValue() bool {
	if ag.Value == nil {
		return false
	}
	stringValue := ag.StringValue()
	if v, err := strconv.ParseBool(stringValue); err == nil {
		return v
	}
	return false
}

// ChannelValue is a utility function for casting value to a channel struct
// Returns a channel struct, partial channel struct, or a nil value
func (ag CommandArg) ChannelValue(s *discordgo.Session) (*discordgo.Channel, error) {
	chanID := ag.StringValue()
	if chanID == "" {
		return &discordgo.Channel{ID: chanID}, errors.New("no channel id")
	}

	if s == nil {
		return &discordgo.Channel{ID: chanID}, errors.New("no session")
	}
	cleanedId := CleanId(chanID)

	if cleanedId == "" {
		return &discordgo.Channel{ID: chanID}, errors.New("not an id")
	}
	ch, err := s.State.Channel(cleanedId)

	if err != nil {
		ch, err = s.Channel(cleanedId)
		if err != nil {
			return &discordgo.Channel{ID: chanID}, errors.New("could not find channel")
		}
	}
	return ch, nil
}

// MemberValue is a utility function for casting value to a member struct
// Returns a user struct, partial user struct, or a nil value
func (ag CommandArg) MemberValue(s *discordgo.Session, g string) (*discordgo.Member, error) {
	userID := ag.StringValue()
	if userID == "" {
		return &discordgo.Member{
			GuildID: g,
			User: &discordgo.User{
				ID: userID,
			},
		}, errors.New("no userid")
	}
	cleanedId := CleanId(userID)
	if cleanedId == "" {
		return &discordgo.Member{
			GuildID: g,
			User: &discordgo.User{
				ID: userID,
			},
		}, errors.New("invalid userid")
	}
	if s == nil {
		return &discordgo.Member{
			GuildID: g,
			User: &discordgo.User{
				ID: cleanedId,
			},
		}, errors.New("session is nil")
	}
	u, err := s.State.Member(g, cleanedId)

	if err != nil {
		u, err = s.GuildMember(g, cleanedId)
		if err != nil {
			return &discordgo.Member{
				GuildID: g,
				User: &discordgo.User{
					ID: userID,
				},
			}, errors.New("cant find user")
		}
	}
	return u, nil
}

// UserValue is a utility function for casting value to a member struct
// Returns a user struct, partial user struct, or a nil value
func (ag CommandArg) UserValue(s *discordgo.Session) (*discordgo.User, error) {
	userID := ag.StringValue()
	if userID == "" {
		return &discordgo.User{
			ID: userID,
		}, errors.New("no userid")
	}
	cleanedId := CleanId(userID)
	if cleanedId == "" {
		return &discordgo.User{
			ID: userID,
		}, errors.New("invalid userid")
	}
	if s == nil {
		return &discordgo.User{
			ID: userID,
		}, errors.New("session is nil")
	}
	u, err := s.User(cleanedId)

	if err != nil {
		return &discordgo.User{

			ID: cleanedId,
		}, errors.New("cant find user")
	}
	return u, nil
}

// RoleValue is a utility function for casting value to a user struct
// Returns a user struct, partial user struct, or a nil value
func (ag CommandArg) RoleValue(s *discordgo.Session, gID string) (*discordgo.Role, error) {
	roleID := ag.StringValue()
	if roleID == "" {
		return nil, errors.New("unable to find roleid")
	}
	cleanedId := CleanId(roleID)
	if cleanedId == "" {
		return &discordgo.Role{
			ID: roleID,
		}, errors.New("invalid roleid")
	}
	if s == nil || gID == "" {
		return &discordgo.Role{ID: cleanedId}, errors.New("no session (and/or) guild id")
	}
	r, err := s.State.Role(cleanedId, gID)

	if err != nil {
		roles, err := s.GuildRoles(gID)
		if err == nil {
			for _, r = range roles {
				if r.ID == cleanedId {
					return r, nil
				}
			}
		}
		return &discordgo.Role{ID: roleID}, errors.New("could not find role")
	}
	return r, nil
}
