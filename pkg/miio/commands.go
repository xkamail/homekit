package miio

// Independent device command.
type deviceCommand struct {
	ID     int64         `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params,omitempty"`
}

// Base response from the device.
type devResponse struct {
	ID int64 `json:"id"`
}
