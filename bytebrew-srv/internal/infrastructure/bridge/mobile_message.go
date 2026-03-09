package bridge

// MobileMessage is the wire format for messages exchanged with mobile devices
// via the bridge WebSocket connection.
type MobileMessage struct {
	Type      string                 `json:"type"`
	RequestID string                 `json:"request_id,omitempty"`
	DeviceID  string                 `json:"device_id,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}
