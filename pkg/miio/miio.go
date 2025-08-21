package miio

import (
	"encoding/json"
	"log/slog"
	"time"
)

type Mi struct {
	XiaomiDevice

	Update chan *DeviceUpdateMessage
	Level  int
	Swing  bool
	Power  bool
	f      func(level int, swing bool, power bool)
}

func New(ip string, token string) (*Mi, error) {
	mi := Mi{
		XiaomiDevice: XiaomiDevice{
			rawState: make(map[string]interface{}),
		},
	}
	err := mi.start(ip, token, defaultPort)
	if err != nil {
		return nil, err
	}

	mi.Update = make(chan *DeviceUpdateMessage, 100)
	go mi.pollStatus()
	return &mi, nil
}

func (m *Mi) OnUpdate(f func(level int, swing bool, power bool)) {
	m.f = f
}

// Stop stops the device.
func (m *Mi) Stop() {
	m.stop()
	close(m.Update)
}

type Properties struct {
	Did   string `json:"did"`
	Siid  int    `json:"siid"`
	Piid  int    `json:"piid"`
	Value any    `json:"value"`
}

func (m *Mi) SetPower(on bool) {
	m.sendCommand("set_properties", []any{
		Properties{
			Siid:  2,
			Piid:  1,
			Value: on,
		},
	}, false, 5)
}

func (m *Mi) SetLevel(n int) {
	m.sendCommand("set_properties", []any{
		Properties{
			Siid:  2,
			Piid:  5,
			Value: n,
		},
	}, false, 5)
}

func (m *Mi) SetSwing(b bool) {
	m.sendCommand("set_properties", []any{
		Properties{
			Siid:  2,
			Piid:  6,
			Value: b,
		},
	}, false, 5)
}

func (m *Mi) GetProperties() bool {
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

func (m *Mi) updateState() {
	if m.rawState["get_properties"] == nil {
		return
	}
	var r Result
	json.Unmarshal(m.rawState["get_properties"].([]byte), &r)
	slog.Info("updateState", "result", r.Result)
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

// leak memory
func (m *Mi) pollStatus() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.GetProperties()
		}
	}
}
