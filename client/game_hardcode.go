package client

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

// GameHardCode хардкод координат
type GameHardCode struct {
	client     *Client
	areasQueue chan ExploreAreaOut
	coinsQueue chan Coin
	licenses   chan SyncLicense
}

func NewGameHardCode(client *Client) *GameHardCode {
	return &GameHardCode{
		client:     client,
		areasQueue: make(chan ExploreAreaOut, 40000),
		coinsQueue: make(chan Coin, 600),
		licenses:   make(chan SyncLicense, 100),
	}
}

func (g *GameHardCode) issueLicense() error {
	var license License
	var err error

	for {
		if len(g.coinsQueue) != 0 {
			license, err = g.client.PostLicenses([]Coin{<-g.coinsQueue})
		} else {
			license, err = g.client.PostLicenses(nil)
		}

		if err != nil {
			return err
		}

		syncLicense := NewSyncLicense(license)

		for i := 0; i < license.DigAllowed; i++ {
			g.licenses <- syncLicense
		}

		syncLicense.WaitLicenseEnd()
	}
}

func (g *GameHardCode) loadAreas() error {
	//filePath := "/home/user/projects/hlc21/hardcode.txt"
	filePath := "/tmp/hlc/app/hardcode.txt"

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		areaRaw := strings.Split(scanner.Text(), ",")

		posX, _ := strconv.Atoi(areaRaw[0])
		posY, _ := strconv.Atoi(areaRaw[1])
		amount, _ := strconv.Atoi(areaRaw[2])

		g.areasQueue <- ExploreAreaOut{
			Area:   ExploreAreaIn{PosX: posX, PosY: posY, SizeX: 1, SizeY: 1},
			Amount: amount,
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (g *GameHardCode) cash(treasure Treasure) error {
	coins, err := g.client.PostCash(treasure)
	if err != nil {
		return err
	}

	if coins != nil && len(g.coinsQueue) < coinsMax {
		for _, coin := range coins {
			g.coinsQueue <- coin
		}
	}

	return nil
}

func (g *GameHardCode) dig() error {
	for column := range g.areasQueue {
		columnDepth := 1
		columnTreasures := 0
		for columnTreasures < column.Amount && columnDepth <= 10 {
			syncLicense := <-g.licenses

			treasures, err := g.client.PostDig(syncLicense.License.ID, column.Area.PosX, column.Area.PosY, columnDepth)
			if err != nil {
				if errors.Is(err, NoTreasureErr) {
					columnDepth++
					syncLicense.DecreaseDig()
					continue
				}

				return err
			}

			if columnDepth >= 3 {
				for _, treasure := range treasures {
					if err := g.cash(treasure); err != nil {
						return err
					}
				}
			}

			columnTreasures += len(treasures)
			columnDepth++
			syncLicense.DecreaseDig()
		}
	}

	return nil
}

func (g *GameHardCode) Run() error {
	if err := g.loadAreas(); err != nil {
		return err
	}

	//****** LICENSES *****
	for i := 0; i < 10; i++ {
		go func() {
			if err := g.issueLicense(); err != nil {
				log.Fatalln(err)
			}
		}()
	}

	wg := new(sync.WaitGroup)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			if err := g.dig(); err != nil {
				log.Fatalln(err)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	return nil
}
