package handler

type Create struct {
	Created bool `json:"created"`
}

type Update struct {
	Updated bool `json:"updated"`
}

type Delete struct {
	Deleted bool `json:"deleted"`
}