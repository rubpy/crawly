package crawly

import (
	"time"

	"github.com/rubpy/crawly/csync"
)

//////////////////////////////////////////////////

type CrawlerSettings struct {
	TrackingOrderTimeout         time.Duration `json:"tracking_order_timeout"`
	MinimumTrackingOrderDelay    time.Duration `json:"minimum_tracking_order_delay"`
	MaximumTrackingOrderAttempts int           `json:"maximum_tracking_order_attempts"`

	TrackingTimeout         time.Duration `json:"tracking_timeout"`
	MinimumTrackingDelay    time.Duration `json:"minimum_tracking_delay"`
	MaximumTrackingAttempts int           `json:"maximum_tracking_attempts"`
}

var DefaultCrawlerSettings = CrawlerSettings{
	TrackingOrderTimeout:         45 * time.Second,
	MinimumTrackingOrderDelay:    10 * time.Second,
	MaximumTrackingOrderAttempts: 3,

	TrackingTimeout:         45 * time.Second,
	MinimumTrackingDelay:    10 * time.Second,
	MaximumTrackingAttempts: 10,
}

type SessionSettings csync.SessionSettings

//////////////////////////////////////////////////

func (cr *Crawler) loadSettings() CrawlerSettings {
	return cr.settings.Load()
}

func (cr *Crawler) setSettings(settings CrawlerSettings) {
	cr.settings.Store(settings)
}

func LoadCrawlerSettings(cr *Crawler) (settings CrawlerSettings) {
	if cr == nil {
		return
	}

	return cr.loadSettings()
}

func SetCrawlerSettings(cr *Crawler, settings CrawlerSettings) {
	if cr == nil {
		return
	}

	cr.setSettings(settings)
}
