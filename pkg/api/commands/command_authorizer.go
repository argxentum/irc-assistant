package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
)

type CommandAuthorizer interface {
	RequiredRole() Role
	RequiredChannelStatus() irc.ChannelStatus
	IsAuthorized(e *irc.Event, channel string, callback func(bool))
	IsUserAuthorizedByRole(nick string, role Role) bool
	IsUserAuthorizedByChannelStatus(e *irc.Event, channel string, status irc.ChannelStatus, callback func(bool))
	GetUser(channel, nick string, callback func(user *irc.User))
	ListUsers(channel string, callback func([]*irc.User))
	ListUsersByMask(channel, mask string, callback func([]*irc.User))
}

type commandAuthorizer struct {
	ctx                   context.Context
	cfg                   *config.Config
	irc                   irc.IRC
	requiredRole          Role
	requiredChannelStatus irc.ChannelStatus
}

func newCommandAuthorizer(ctx context.Context, cfg *config.Config, irc irc.IRC, role Role, channelStatus irc.ChannelStatus) *commandAuthorizer {
	return &commandAuthorizer{
		ctx:                   ctx,
		cfg:                   cfg,
		irc:                   irc,
		requiredRole:          role,
		requiredChannelStatus: channelStatus,
	}
}

func (c *commandAuthorizer) RequiredRole() Role {
	return c.requiredRole
}

func (c *commandAuthorizer) RequiredChannelStatus() irc.ChannelStatus {
	return c.requiredChannelStatus
}

// IsUserAuthorizedByRole checks if the given sender is authorized based on authorization configuration settings
func (c *commandAuthorizer) IsUserAuthorizedByRole(nick string, role Role) bool {
	switch role {
	case RoleOwner:
		return nick == c.cfg.IRC.Owner
	case RoleAdmin:
		if nick == c.cfg.IRC.Owner {
			return true
		}
		for _, a := range c.cfg.IRC.Admins {
			if nick == a {
				return true
			}
		}
		return false
	}
	return true
}

// GetUser retrieves a user in the channel by nick
func (c *commandAuthorizer) GetUser(channel, nick string, callback func(user *irc.User)) {
	c.irc.GetUser(channel, nick, callback)
}

func (c *commandAuthorizer) ListUsers(channel string, callback func([]*irc.User)) {
	c.irc.ListUsers(channel, callback)
}

func (c *commandAuthorizer) ListUsersByMask(channel, mask string, callback func([]*irc.User)) {
	c.irc.ListUsersByMask(channel, mask, callback)
}

// IsUserAuthorizedByChannelStatus checks if the given sender is authorized based on their channel status
func (c *commandAuthorizer) IsUserAuthorizedByChannelStatus(e *irc.Event, channel string, status irc.ChannelStatus, callback func(bool)) {
	nick, _ := e.Sender()

	c.ListUsers(channel, func(users []*irc.User) {
		for _, user := range users {
			if user.Mask.Nick == nick && irc.IsStatusAtLeast(user.Status, status) {
				callback(true)
				return
			}
		}
		callback(false)
	})
}

// IsAuthorized checks authorization using both channel status-based and role-based authorization
func (c *commandAuthorizer) IsAuthorized(e *irc.Event, channel string, callback func(bool)) {
	if len(c.requiredChannelStatus) > 0 {
		c.IsUserAuthorizedByChannelStatus(e, channel, c.requiredChannelStatus, func(authorized bool) {
			if authorized {
				callback(true)
				return
			}

			if len(c.requiredRole) > 0 {
				nick, _ := e.Sender()
				if c.IsUserAuthorizedByRole(nick, c.requiredRole) {
					callback(true)
					return
				}

				callback(false)
				return
			}

			callback(false)
		})
	} else if len(c.requiredRole) > 0 {
		nick, _ := e.Sender()
		if c.IsUserAuthorizedByRole(nick, c.requiredRole) {
			callback(true)
			return
		}

		callback(false)
	} else {
		callback(true)
	}
}
