package miio

import (
	"encoding/json"
	"time"
)

type AirPurifier struct {
	XiaomiDevice
	f       func(mode int, quality int)
	quality int
	// 0 = auto, 1 = sleep, 2 = favorite
	mode int
}

func NewAirPurifier(ip string, token string) (*AirPurifier, error) {
	mi := AirPurifier{
		XiaomiDevice: XiaomiDevice{
			rawState: make(map[string]interface{}),
		},
	}
	err := mi.start(ip, token, defaultPort)
	if err != nil {
		return nil, err
	}

	go mi.pollStatus()
	return &mi, nil
}

func (p *AirPurifier) pollStatus() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			p.GetProperties()
			if p.f != nil {
				p.f(p.mode, p.quality)
			}

		}
	}
}

func (p *AirPurifier) OnUpdate(f func(mode int, quality int)) {
	p.f = f
}

// SetMode mode 0 = auto, 1 = sleep, 2 = favorite
func (p *AirPurifier) SetMode(mode int) {
	p.sendCommand("set_properties", []any{
		Properties{
			Siid:  2,
			Piid:  4,
			Value: mode,
		},
	}, false, 5)
}

func (p *AirPurifier) GetProperties() {
	p.sendCommand("get_properties", []any{
		Properties{
			Siid: 3,
			Piid: 4,
		},
		Properties{
			Siid: 2,
			Piid: 4,
		},
	}, true, 5)
	var result Result
	json.Unmarshal(p.rawState["get_properties"].([]byte), &result)
	p.Lock()
	for _, v := range result.Result {
		if v.Siid == 3 && v.Piid == 4 {
			if val, ok := v.Value.(float64); ok {
				p.quality = int(val)
			}
		}
		if v.Siid == 2 && v.Piid == 4 {
			if val, ok := v.Value.(float64); ok {
				p.mode = int(val)
			}
		}
	}
	p.Unlock()

}
