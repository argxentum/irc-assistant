package modes

import (
	"fmt"
	"sync"
	"time"
)

var manager *ChannelModeManager

type ChannelModeManager struct {
	sync.RWMutex
	modes     map[string]ChannelMode
	timers    map[string]*time.Timer
	cooldowns map[string]map[string]time.Time
}

func GetManager() *ChannelModeManager {
	if manager == nil {
		manager = &ChannelModeManager{
			modes:     make(map[string]ChannelMode),
			timers:    make(map[string]*time.Timer),
			cooldowns: make(map[string]map[string]time.Time),
		}
	}
	return manager
}

func (m *ChannelModeManager) Activate(mode ChannelMode, cooldown time.Duration) error {
	m.Lock()
	defer m.Unlock()

	channel := mode.Channel()

	if _, ok := m.modes[channel]; ok {
		return fmt.Errorf("a mode is already active in %s", channel)
	}

	if cooldown > 0 {
		if channelCooldowns, ok := m.cooldowns[channel]; ok {
			if last, ok := channelCooldowns[mode.Name()]; ok {
				remaining := cooldown - time.Since(last)
				if remaining > 0 {
					return fmt.Errorf("cooldown active, try again in %s", remaining.Round(time.Second))
				}
			}
		}
	}

	m.modes[channel] = mode

	if timeout := mode.Timeout(); timeout > 0 {
		m.timers[channel] = time.AfterFunc(timeout, func() {
			m.Deactivate(channel)
		})
	}

	go mode.OnStart()

	return nil
}

func (m *ChannelModeManager) Deactivate(channel string) {
	m.Lock()

	mode, ok := m.modes[channel]
	if !ok {
		m.Unlock()
		return
	}

	if timer, ok := m.timers[channel]; ok {
		timer.Stop()
		delete(m.timers, channel)
	}

	if m.cooldowns[channel] == nil {
		m.cooldowns[channel] = make(map[string]time.Time)
	}
	m.cooldowns[channel][mode.Name()] = time.Now()

	delete(m.modes, channel)

	m.Unlock()

	mode.OnEnd()
}

func (m *ChannelModeManager) ActiveMode(channel string) ChannelMode {
	m.RLock()
	defer m.RUnlock()
	return m.modes[channel]
}

func (m *ChannelModeManager) IsActive(channel string) bool {
	m.RLock()
	defer m.RUnlock()
	_, ok := m.modes[channel]
	return ok
}

func (m *ChannelModeManager) CooldownRemaining(channel, modeName string, cooldown time.Duration) time.Duration {
	m.RLock()
	defer m.RUnlock()

	if channelCooldowns, ok := m.cooldowns[channel]; ok {
		if last, ok := channelCooldowns[modeName]; ok {
			remaining := cooldown - time.Since(last)
			if remaining > 0 {
				return remaining
			}
		}
	}
	return 0
}
