package protocol

type MessageType string

const (
	MessageTypeLoginRequest  MessageType = "login_request"
	MessageTypeLoginResponse MessageType = "login_response"
)

type LoginRequest struct {
	Type          MessageType `json:"type"`
	Token         string      `json:"token"`
	ClientVersion string      `json:"client_version"`
	OS            string      `json:"os"`
}

type LoginResponse struct {
	Type        MessageType `json:"type"`
	Status      string      `json:"status"` // "success" or "error"
	AssignedVIP string      `json:"assigned_vip,omitempty"`
	SubnetMask  string      `json:"subnet_mask,omitempty"`
	ServerVIP   string      `json:"server_vip,omitempty"`
	Message     string      `json:"message,omitempty"`
}
