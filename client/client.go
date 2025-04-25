package client

// Client abstract device and user
type Client interface {
	WriteMsg(typ int, data []byte)

	// For users, return the device ID that the user wants to access
	// For devices, return the ID of the device
	DeviceID() string

	IsDevice() bool

	Close()

	CloseConn()

	Closed() bool
}
