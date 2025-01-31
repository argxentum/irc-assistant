package irc

import (
	"fmt"
	"strings"
)

type Mask struct {
	Nick   string
	UserID string
	Host   string
}

func (m *Mask) String() string {
	n := m.Nick
	if len(n) == 0 {
		n = "*"
	}

	u := m.UserID
	if len(u) == 0 {
		u = "*"
	}

	h := m.Host
	if len(h) == 0 {
		h = "*"
	}

	return fmt.Sprintf("%s!%s@%s", n, u, h)
}

func (m *Mask) NickWildcardString() string {
	return fmt.Sprintf("*!%s@%s", m.UserID, m.Host)
}

func ParseMask(mask string) *Mask {
	n := strings.Split(mask, "!")
	if len(n) != 2 {
		return nil
	}

	u := strings.Split(n[1], "@")
	if len(u) != 2 {
		return nil
	}

	return &Mask{
		Nick:   n[0],
		UserID: u[0],
		Host:   u[1],
	}
}
