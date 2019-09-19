package message

// Message format
type Message struct {
	Command string      `json:"cmd"`
	SN      uint32      `json:"sn"`
	Data    interface{} `json:"data"`
	Mode    int32       `json:"mode"`
	Channel Channel     `json:"channel"`
	Group   Group       `json:"group"`
	User    User        `json:"user"`
	Target  string      `json:"target"`
}

// Channel format
type Channel struct {
	ID   string `json:"id"`
	Stat int32  `json:"stat"`
}

// Group format
type Group struct {
	ID   int32 `json:"id"`
	Stat int32 `json:"stat"`
}

// User format
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role int32  `json:"role"`
	Icon string `json:"icon"`
}
