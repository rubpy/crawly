package crawly

import "time"

//////////////////////////////////////////////////

type Result struct {
	Err   error `json:"err"`
	Valid bool  `json:"valid"`
	Idle  bool  `json:"idle"`

	SessionID string    `json:"session_id"`
	Pass      uint64    `json:"pass"`
	Timestamp time.Time `json:"timestamp"`

	Orders   map[Handle]TrackingResult `json:"orders"`
	Entities map[Handle]TrackingResult `json:"entities"`
}
