package tf2pickup

type Avatar struct {
	Small string `json:"small"`
}

type Player struct {
	Name           string `json:"name"`
	Avatar         Avatar `json:"avatar"`
	SteamId        int64  `json:"steamId,string"`
	Etf2LProfileId int    `json:"etf2lProfileId"`
}

type Slot struct {
	Player    Player `json:"player"`
	Team      string `json:"team"`
	GameClass string `json:"gameClass"`
}

type Result struct {
	Id     string `json:"id"`
	Number int64  `json:"number"`
	Slots  []Slot `json:"slots"`
	State  string `json:"state"`
	Score  Score  `json:"score"`
}

type Score struct {
	Blu int64 `json:"blu"`
	Red int64 `json:"red"`
}
