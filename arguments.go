package framework

import (
	"errors"
	"github.com/QPixel/orderedmap"
	"github.com/bwmarrin/discordgo"
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
	Boolean   ArgTypeGuards = "bool"
	SubCmd    ArgTypeGuards = "subcmd"
	SubCmdGrp ArgTypeGuards = "subcmdgrp"
	ArrString ArgTypeGuards = "arrString"
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
	Regex         string
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
func CreateCommandInfo(trigger string, description string, public bool, group string) *CommandInfo {
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
		Regex:         "",
	})
	return cI
}

// AddFlagArg
// Adds a flag arg, which is a special type of argument
func (cI *CommandInfo) AddFlagArg(flag string, typeGuard ArgTypeGuards, match ArgTypes, description string, required bool, defaultOption string) *CommandInfo {
	cI.Arguments.Set(flag, &ArgInfo{
		Description:   description,
		Required:      required,
		Flag:          true,
		Match:         match,
		TypeGuard:     typeGuard,
		DefaultOption: defaultOption,
		Regex:         "",
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

// -- Argument Parser --

// ParseArguments
// Parses the arguments into a pointer to an Arguments struct
func ParseArguments(args string, infoArgs *orderedmap.OrderedMap) *Arguments {
	ar := make(Arguments)
	if args == "" || len(infoArgs.Keys()) < 1 {
		return &ar
	}
	// Split string on spaces to get every "phrase"
	splitString := strings.Split(args, " ")

	// Current Position in the infoArgs map
	currentPos := 0

	// Keys of infoArgs
	k := infoArgs.Keys()

	for i := 0; i < len(splitString); i++ {
		for n := currentPos; n <= len(k); n++ {
			if n > len(k)+1 || currentPos+1 > len(k) {
				break
			}
			v, _ := infoArgs.Get(k[currentPos])
			vv := v.(*ArgInfo)
			switch vv.Match {
			case ArgOption:
				// Lets first check the typeguard to see if the str matches the arg
				if checkTypeGuard(splitString[i], vv.TypeGuard) {
					// todo abstract this into handleArgOption
					// Handle quoted ArgOptions separately
					if strings.Contains(splitString[i], "\"") {
						st := CommandArg{}
						st, i = handleQuotedString(splitString, *vv, i)
						ar[k[currentPos]] = st
						currentPos++
						break
					}
					// Handle ArgOption
					ar[k[currentPos]] = handleArgOption(splitString[i], *vv)
					currentPos++
					break
				}
				if n+1 > len(splitString) {
					break
				}
				// If the TypeGuard does not match check to see if the Arg is required or not
				if vv.Required {
					// Set the CommandArg to the default option, which is usually ""
					ar[k[currentPos]] = CommandArg{
						info:  *vv,
						Value: vv.DefaultOption,
					}
					currentPos++
					break
				} else {
					// If it's not required, we set the CommandArg to ""
					ar[k[currentPos]] = CommandArg{
						info:  *vv,
						Value: "",
					}
					currentPos++
					break
				}
			case ArgContent:
				// Takes the splitString and currentPos to find how many more elements in the slice
				// need to join together
				contentString := ""
				contentString, i = createContentString(splitString, i)
				ar[k[currentPos]] = CommandArg{
					info:  *vv,
					Value: contentString,
				}
				break
			default:
				break
			}
			continue
		}
	}
	return &ar
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

func handleQuotedString(splitString []string, argInfo ArgInfo, currentPos int) (CommandArg, int) {
	str := ""
	splitString[currentPos] = strings.TrimPrefix(splitString[currentPos], "\"")
	for i := currentPos; i < len(splitString); i++ {
		if !strings.HasSuffix(splitString[i], "\"") {
			str += splitString[i] + " "
		} else {
			str += strings.TrimSuffix(splitString[i], "\"")
			currentPos = i
			break
		}
	}
	return CommandArg{
		info:  argInfo,
		Value: str,
	}, currentPos
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
