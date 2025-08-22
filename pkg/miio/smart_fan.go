package miio

import (
	"encoding/json"
	"time"
)

type SmartFan struct {
	XiaomiDevice

	Level int
	Swing bool
	Power bool
	f     func(level int, swing bool, power bool)
}

func NewSmartFan(ip string, token string) (*SmartFan, error) {
	mi := SmartFan{
		XiaomiDevice: XiaomiDevice{
			rawState: make(map[string]interface{}),
		},
	}
	err := mi.start(ip, token, defaultPort)
	if err != nil {
		return nil, err
	}
	return &mi, nil
}

func (m *SmartFan) OnUpdate(f func(level int, swing bool, power bool)) {
	m.f = f
}

// Stop stops the device.
func (m *SmartFan) Stop() {
	m.stop()
}

type Properties struct {
	Did   string `json:"did"`
	Siid  int    `json:"siid"`
	Piid  int    `json:"piid"`
	Value any    `json:"value"`
}

func (m *SmartFan) SetPower(on bool) {
	m.sendCommand("set_properties", []any{
		Properties{
			Siid:  2,
			Piid:  1,
			Value: on,
		},
	}, false, 5)
}

func (m *SmartFan) SetLevel(n int) {
	m.sendCommand("set_properties", []any{
		Properties{
			Siid:  2,
			Piid:  5,
			Value: n,
		},
	}, false, 5)
}

func (m *SmartFan) SetSwing(b bool) {
	m.sendCommand("set_properties", []any{
		Properties{
			Siid:  2,
			Piid:  6,
			Value: b,
		},
	}, false, 5)
}

func (m *SmartFan) GetProperties() bool {
	m.sendCommand("get_properties", []any{
		Properties{
			Siid: 2,
			Piid: 1,
		}, // power
		Properties{
			Siid: 2,
			Piid: 5,
		}, // speed
		Properties{
			Siid: 2,
			Piid: 6,
		}, // swing ?
	}, true, 1)
	m.updateState()
	return false
}

// {\"id\":1755746239,\"result\":[{\"did\":\"\",\"siid\":2,\"piid\":1,\"code\":0,\"value\":true},{\"did\":\"\",\"siid\":2,\"piid\":5,\"code\":0,\"value\":62},{\"did\":\"\",\"siid\":2,\"piid\":6,\"code\":0,\"value\":false}],\"exe_time\":142}

type Result struct {
	ID     int          `json:"id"`
	Result []Properties `json:"result"`
}

func (m *SmartFan) updateState() {
	if m.rawState["get_properties"] == nil {
		return
	}
	var r Result
	json.Unmarshal(m.rawState["get_properties"].([]byte), &r)
	m.Lock()
	for _, p := range r.Result {
		switch p.Piid {
		case 1:
			if v, ok := p.Value.(bool); ok {
				m.Power = v
			}
		case 5:
			if f, ok := p.Value.(float64); ok {
				m.Level = int(f)
			}
		case 6:
			if v, ok := p.Value.(bool); ok {
				m.Swing = v
			}
		}
	}
	m.Unlock()

	if m.f != nil {
		m.f(m.Level, m.Swing, m.Power)
	}
}

// PollStatus polls the device status.
func (m *SmartFan) PollStatus(tick <-chan time.Time) {
	for {
		select {
		case <-tick:
			m.GetProperties()
		}
	}
}
