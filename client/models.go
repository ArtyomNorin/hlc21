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

type DigIn struct {
	LicenseID int `json:"licenseID"`
	PosX      int `json:"posX"`
	PosY      int `json:"posY"`
	Depth     int `json:"depth"`
}

type Coin int
