package penalty

import "assistant/pkg/config"

type Status struct {
	MutePercent float64
	BanPercent  float64
}

func Calculate(penalty, extendedPenalty int, cfg config.DisinfoPenaltyConfig) Status {
	var s Status

	if cfg.TempMuteThreshold > 0 {
		s.MutePercent = float64(penalty) / float64(cfg.TempMuteThreshold) * 100
		if s.MutePercent < 0 {
			s.MutePercent = 0
		}
		if s.MutePercent > 100 {
			s.MutePercent = 100
		}
	}

	if cfg.TempBanThreshold > 0 {
		s.BanPercent = float64(extendedPenalty) / float64(cfg.TempBanThreshold) * 100
		if s.BanPercent < 0 {
			s.BanPercent = 0
		}
		if s.BanPercent > 100 {
			s.BanPercent = 100
		}
	}

	return s
}
