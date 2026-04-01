package server

import "github.com/stockyard-dev/stockyard-paddock/internal/license"

// Limits holds the feature limits for the current license tier.
type Limits struct {
	MaxMonitors      int  // 0 = unlimited (Pro)
	MinIntervalSec   int  // minimum check interval in seconds
	RetentionDays    int  // 7 free, 90 pro
	AlertWebhooks    bool // Pro only
	SSLExpiry        bool // Pro only
	CustomStatusPage bool // Pro only
}

var freeLimits = Limits{
	MaxMonitors:      3,
	MinIntervalSec:   300, // 5 minutes
	RetentionDays:    7,
	AlertWebhooks:    false,
	SSLExpiry:        false,
	CustomStatusPage: false,
}

var proLimits = Limits{
	MaxMonitors:      0,
	MinIntervalSec:   30,
	RetentionDays:    90,
	AlertWebhooks:    true,
	SSLExpiry:        true,
	CustomStatusPage: true,
}

func LimitsFor(info *license.Info) Limits {
	if info != nil && info.IsPro() {
		return proLimits
	}
	return freeLimits
}

func LimitReached(limit, current int) bool {
	if limit == 0 {
		return false
	}
	return current >= limit
}
