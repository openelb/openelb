package net

import (
	"encoding/json"
	"net"
)

// Sub class net.HardwareAddr so that we can add JSON marshalling and unmarshalling.
type MAC struct {
	net.HardwareAddr
}

// MarshalJSON interface for a MAC
func (m MAC) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

// UnmarshalJSON interface for a MAC
func (m *MAC) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if mac, err := net.ParseMAC(s); err != nil {
		return err
	} else {
		m.HardwareAddr = mac
		return nil
	}
}
