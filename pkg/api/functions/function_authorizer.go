package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
)

type FunctionAuthorizer interface {
	RequiredRole() Role
	RequiredChannelStatus() irc.ChannelStatus
	IsAuthorized(e *irc.Event, channel string, callback func(bool))
	IsUserAuthorizedByRole(nick string, role Role) bool
	IsUserAuthorizedByChannelStatus(e *irc.Event, channel string, status irc.ChannelStatus, callback func(bool))
	UserStatus(channel, nick string, callback func(user *irc.User))
	UserStatuses(channel string, callback func([]irc.User))
}

type functionAuthorizer struct {
	ctx                   context.Context
	cfg                   *config.Config
	irc                   irc.IRC
	requiredRole          Role
	requiredChannelStatus irc.ChannelStatus
}

func newFunctionAuthorizer(ctx context.Context, cfg *config.Config, irc irc.IRC, role Role, channelStatus irc.ChannelStatus) *functionAuthorizer {
	return &functionAuthorizer{
		ctx:                   ctx,
		cfg:                   cfg,
		irc:                   irc,
		requiredRole:          role,
		requiredChannelStatus: channelStatus,
	}
}

func (f *functionAuthorizer) RequiredRole() Role {
	return f.requiredRole
}

func (f *functionAuthorizer) RequiredChannelStatus() irc.ChannelStatus {
	return f.requiredChannelStatus
}

// IsUserAuthorizedByRole checks if the given sender is authorized based on authorization configuration settings
func (f *functionAuthorizer) IsUserAuthorizedByRole(nick string, role Role) bool {
	switch role {
	case RoleOwner:
		return nick == f.cfg.IRC.Owner
	case RoleAdmin:
		if nick == f.cfg.IRC.Owner {
			return true
		}
		for _, a := range f.cfg.IRC.Admins {
			if nick == a {
				return true
			}
		}
		return false
	}
	return true
}

// UserStatus retrieves the user's status in the channel (e.g., operator, half-operator, etc.)
func (f *functionAuthorizer) UserStatus(channel, nick string, callback func(user *irc.User)) {
	f.irc.GetUser(channel, nick, callback)
}

func (f *functionAuthorizer) UserStatuses(channel string, callback func([]irc.User)) {
	f.irc.GetUsers(channel, callback)
}

// IsUserAuthorizedByChannelStatus checks if the given sender is authorized based on their channel status
func (f *functionAuthorizer) IsUserAuthorizedByChannelStatus(e *irc.Event, channel string, status irc.ChannelStatus, callback func(bool)) {
	nick, _ := e.Sender()

	f.UserStatus(channel, nick, func(user *irc.User) {
		if user != nil && !irc.IsStatusAtLeast(user.Status, status) {
			callback(false)
			return
		}
		callback(true)
	})
}

// IsAuthorized checks authorization using both channel status-based and role-based authorization
func (f *functionAuthorizer) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	if len(f.requiredChannelStatus) > 0 {
		f.IsUserAuthorizedByChannelStatus(e, channel, f.requiredChannelStatus, func(authorized bool) {
			if authorized {
				callback(true)
				return
			}

			if len(f.requiredRole) > 0 {
				nick, _ := e.Sender()
				if f.IsUserAuthorizedByRole(nick, f.requiredRole) {
					callback(true)
					return
				}

				callback(false)
				return
			}

			callback(false)
		})
	} else if len(f.requiredRole) > 0 {
		nick, _ := e.Sender()
		if f.IsUserAuthorizedByRole(nick, f.requiredRole) {
			callback(true)
			return
		}

		callback(false)
	} else {
		callback(true)
	}
}
