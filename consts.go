package framework

import "github.com/dlclark/regexp2"

type regex map[string]*regexp2.Regexp

var (
	TimeRegexes = regex{
		"seconds": regexp2.MustCompile("^[0-9]+s$", 0),
		"minutes": regexp2.MustCompile("^[0-9]+m$", 0),
		"hours":   regexp2.MustCompile("^[0-9]+h$", 0),
		"days":    regexp2.MustCompile("^[0-9]+d$", 0),
		"weeks":   regexp2.MustCompile("^[0-9]+w$", 0),
		"years":   regexp2.MustCompile("^[0-9]+y$", 0),
	}
	MentionStringRegexes = regex{
		"all":     regexp2.MustCompile("<((@!?\\d+)|(#?\\d+)|(@&?\\d+))>", 0),
		"role":    regexp2.MustCompile("<((@&?\\d+))>", 0),
		"user":    regexp2.MustCompile("<((@!?\\d+))>", 0),
		"channel": regexp2.MustCompile("<((#?\\d+))>", 0),
		"id":      regexp2.MustCompile("^[0-9]{18}$", 0),
	}
	TypeGuard = regex{
		"message_url": regexp2.MustCompile("((https:\\/\\/canary.discord.com\\/channels\\/)+([0-9]{18})\\/+([0-9]{18})\\/+([0-9]{18})$)", regexp2.IgnoreCase|regexp2.Multiline),
	}
	Misc = regex{
		"quoted_string": regexp2.MustCompile("^\"[a-zA-Z0-9]+\"$", 0),
		"int":           regexp2.MustCompile("\\b(0*(?:[0-9]{1,8}))\\b", 0),
	}
)
