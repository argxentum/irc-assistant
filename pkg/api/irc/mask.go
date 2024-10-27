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

func (h *Mask) String() string {
	return fmt.Sprintf("%s!%s@%s", h.Nick, h.UserID, h.Host)
}

func (h *Mask) NickWildcardString() string {
	return fmt.Sprintf("*!%s@%s", h.UserID, h.Host)
}

func Parse(mask string) *Mask {
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
