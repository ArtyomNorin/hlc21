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

func (area ExploreAreaOut) GetRightArea(leftArea ExploreAreaOut) ExploreAreaOut {
	return ExploreAreaOut{
		Area: ExploreAreaIn{
			PosX:  area.Area.PosX + leftArea.Area.SizeX,
			PosY:  area.Area.PosY,
			SizeX: area.Area.SizeX - leftArea.Area.SizeX,
			SizeY: area.Area.SizeY,
		},
		Amount: area.Amount - leftArea.Amount,
	}
}

type License struct {
	ID         int `json:"id"`
	DigAllowed int `json:"digAllowed"`
	DigUsed    int `json:"digUsed"`
}

func (l License) IsEnd() bool {
	return l.DigUsed >= l.DigAllowed
}

func (l License) IsFree() bool {
	return l.DigAllowed == 3
}

type SyncLicense struct {
	License      License
	endSemaphore chan struct{}
}

func NewSyncLicense(license License) SyncLicense {
	return SyncLicense{
		License:      license,
		endSemaphore: make(chan struct{}, license.DigAllowed),
	}
}

func (sl SyncLicense) DecreaseDig() {
	sl.License.DigUsed++
	sl.endSemaphore <- struct{}{}
}

func (sl SyncLicense) WaitLicenseEnd() {
	for i := 0; i < sl.License.DigAllowed; i++ {
		<-sl.endSemaphore
	}
}

type Treasure string

type PriorityTreasure struct {
	Treasure Treasure
	Priority int
	Index    int
}

type DigIn struct {
	LicenseID int `json:"licenseID"`
	PosX      int `json:"posX"`
	PosY      int `json:"posY"`
	Depth     int `json:"depth"`
}

type Coin int
