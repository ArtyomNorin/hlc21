package client

type ExploreAreaIn struct {
	PosX  int `json:"posX"`
	PosY  int `json:"posY"`
	SizeX int `json:"sizeX"`
	SizeY int `json:"sizeY"`
}

type ExploreAreaOut struct {
	Area   ExploreAreaIn `json:"area"`
	Amount int           `json:"amount"` // кол-во сокровищ в области
}

type License struct {
	ID         int `json:"id"`
	DigAllowed int `json:"digAllowed"`
	DigUsed    int `json:"digUsed"`
}

func (l License) IsEnd() bool {
	return l.DigUsed >= l.DigAllowed
}

type Treasures []string

type DigIn struct {
	LicenseID int `json:"licenseID"`
	PosX      int `json:"posX"`
	PosY      int `json:"posY"`
	Depth     int `json:"depth"`
}
