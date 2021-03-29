package client

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type GameSeq struct {
	gameStats  *GameStats
	client     *Client
	coinsQueue chan Coin
	licenses   chan SyncLicense
}

func NewGameSeq(client *Client) *GameSeq {
	return &GameSeq{
		gameStats: &GameStats{
			exploreCostLock: new(sync.Mutex),
			digCostLock:     new(sync.Mutex),
			cashCostLock:    new(sync.Mutex),
			licenseWaitLock: new(sync.Mutex),
		},
		client:     client,
		coinsQueue: make(chan Coin, 600),
		licenses:   make(chan SyncLicense, 100),
	}
}

func (g *GameSeq) Run() error {
	exploreType := SequenceExploreType
	countThreads := 10
	exploreChunk := 60 //28 bin
	skipLevels := 2

	log.Printf("GameSeq. %s %dx1. %d thread exp->dig->cash loop. %d lvls skip\n", exploreType, exploreChunk, countThreads, skipLevels)

	go func() {
		time.Sleep(time.Second * 598)
		g.gameStats.printStats()
		fmt.Println("----------------------")
		g.client.printStats()
	}()

	// *** LICENSE ***
	for i := 0; i < 10; i++ {
		go func() {
			if err := g.issueLicense(); err != nil {
				log.Fatalln(err)
			}
		}()
	}

	wg := new(sync.WaitGroup)

	yChunkSize := 3500 / countThreads

	for i := 0; i < countThreads; i++ {
		wg.Add(1)
		go func(yStart, yEnd int) {
			columns := make([]ExploreAreaOut, 0, 500)

			for y := yStart; y < yEnd; y++ {
				for x := 0; x < 3500-exploreChunk; x += exploreChunk {
					switch exploreType {
					case SequenceExploreType:
						if err := g.sequenceExplore(&columns, x, y, exploreChunk); err != nil {
							log.Fatalln(err)
						}
					case BinExploreType:
						if err := g.binExplore(&columns, x, y, exploreChunk); err != nil {
							log.Fatalln(err)
						}
					case RandomSequenceExploreType:
						if err := g.randomSequenceExplore(&columns, x, y, exploreChunk, 35); err != nil {
							log.Fatalln(err)
						}
					}

					if len(columns) == 0 {
						continue
					}

					for _, column := range columns {
						if err := g.dig(column, skipLevels); err != nil {
							log.Fatalln(err)
						}
					}

					columns = columns[:0]
				}
			}

			wg.Done()
		}(i*yChunkSize, i*yChunkSize+yChunkSize)
	}

	wg.Wait()

	g.gameStats.printStats()
	fmt.Println("----------------------")
	g.client.printStats()

	return nil
}

func (g *GameSeq) dig(column ExploreAreaOut, skipLevels int) error {
	columnDepth := 1
	columnTreasures := 0

	for columnTreasures < column.Amount {
		startTime := time.Now()
		syncLicense := <-g.licenses
		g.gameStats.writeLicenseWait(time.Since(startTime))

		treasures, err := g.client.PostDig(syncLicense.License.ID, column.Area.PosX, column.Area.PosY, columnDepth)
		if err != nil {
			if errors.Is(err, NoTreasureErr) {
				columnDepth++
				syncLicense.DecreaseDig()
				g.gameStats.writeDigCost(columnDepth)
				continue
			}

			return err
		}

		g.gameStats.writeDigCost(columnDepth)

		if columnDepth > skipLevels {
			for _, treasure := range treasures {
				if err := g.cash(treasure); err != nil {
					return err
				}
			}
		}

		columnTreasures += len(treasures)
		columnDepth++
		syncLicense.DecreaseDig()

		atomic.AddUint64(&g.gameStats.TreasuresFound, uint64(len(treasures)))
	}

	return nil
}

func (g *GameSeq) cash(treasure Treasure) error {
	coins, err := g.client.PostCash(treasure)
	if err != nil {
		return err
	}

	g.gameStats.writeCashCost()

	if coins == nil {
		atomic.AddUint64(&g.gameStats.TreasuresExchangeFailed, 1)
	} else {
		atomic.AddUint64(&g.gameStats.TreasuresExchanged, 1)
	}

	if coins != nil && len(g.coinsQueue) < coinsMax {
		for _, coin := range coins {
			g.coinsQueue <- coin
		}
	}

	return nil
}

func (g *GameSeq) binExplore(columns *[]ExploreAreaOut, x, y, chunkSize int) error {
	area, err := g.client.PostExplore(x, y, chunkSize, 1)
	if err != nil {
		return err
	}

	g.gameStats.writeExploreCost(chunkSize)

	if area.Amount >= 1 {
		if err := g.binExploreRow(columns, area); err != nil {
			return err
		}
	}

	return nil
}

func (g *GameSeq) randomSequenceExplore(columns *[]ExploreAreaOut, x, y, chunkSize, randomAreaSize int) error {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))

	max := x + randomAreaSize - chunkSize
	x = randomizer.Intn(max-x) + x

	area, err := g.client.PostExplore(x, y, chunkSize, 1)
	if err != nil {
		return err
	}

	if area.Amount >= 1 {
		return g.sequenceExploreRow(columns, area, 3)
	}

	return nil
}

func (g *GameSeq) binExploreRow(columns *[]ExploreAreaOut, area ExploreAreaOut) error {
	if area.Amount == 0 {
		return nil
	}

	leftArea, err := g.client.PostExplore(area.Area.PosX, area.Area.PosY, area.Area.SizeX/2, area.Area.SizeY)
	if err != nil {
		return err
	}

	g.gameStats.writeExploreCost(area.Area.SizeX / 2)

	// если ячейка не 1x1
	if leftArea.Area.SizeX != 1 {
		// если все сокровища в левой части, то исследуем только левую часть
		if leftArea.Amount == area.Amount {
			return g.binExploreRow(columns, leftArea)
		}

		// если все сокровища в правой части,  то исследуем только правую часть
		if leftArea.Amount == 0 {
			rightArea := area.GetRightArea(leftArea)
			return g.binExploreRow(columns, rightArea)
		}

		// если сокровища и в левой и в правой части, то исследуем обе части
		if err := g.binExploreRow(columns, leftArea); err != nil {
			return err
		}

		rightArea := area.GetRightArea(leftArea)
		return g.binExploreRow(columns, rightArea)
	}

	atomic.AddUint64(&g.gameStats.SingleCellExplores, 1)

	// если ячейка 1x1

	// если все сокровища в левой ячейке, то сохраняем ячейку
	if leftArea.Amount == area.Amount {
		*columns = append(*columns, leftArea)
		atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)
		return nil
	}

	// если в левой ячейке есть сокровище, то сохраняем её
	if leftArea.Amount > 0 {
		*columns = append(*columns, leftArea)
		atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)
	}

	if area.Area.SizeX%2 != 0 {
		rightArea := area.GetRightArea(leftArea)
		return g.binExploreRow(columns, rightArea)
	}

	rightArea := area.GetRightArea(leftArea)
	*columns = append(*columns, rightArea)
	atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)

	return nil
}

func (g *GameSeq) sequenceExploreRow(columns *[]ExploreAreaOut, area ExploreAreaOut, chunkSize int) error {
	extractedAreaTreasures := 0

	for xArea := area.Area.PosX; xArea < area.Area.PosX+area.Area.SizeX; xArea += chunkSize {
		if extractedAreaTreasures == area.Amount {
			return nil
		} else if xArea+chunkSize == area.Area.PosX+area.Area.SizeX {
			lastArea := ExploreAreaOut{
				Area:   ExploreAreaIn{PosX: xArea, PosY: area.Area.PosY, SizeX: chunkSize, SizeY: area.Area.SizeY},
				Amount: area.Amount - extractedAreaTreasures,
			}

			return g.sequenceExploreBySingleCell(columns, lastArea)
		}

		subArea, err := g.client.PostExplore(xArea, area.Area.PosY, chunkSize, 1)
		if err != nil {
			return err
		}

		g.gameStats.writeExploreCost(chunkSize)

		if subArea.Amount < 1 {
			continue
		}

		if err := g.sequenceExploreBySingleCell(columns, subArea); err != nil {
			return err
		}

		extractedAreaTreasures += subArea.Amount
	}

	return nil
}

func (g *GameSeq) sequenceExplore(columns *[]ExploreAreaOut, x, y, chunkSize int) error {
	area, err := g.client.PostExplore(x, y, chunkSize, 1)
	if err != nil {
		return err
	}

	g.gameStats.writeExploreCost(chunkSize)

	if area.Amount >= 1 {
		return g.sequenceExploreRow(columns, area, 3)
	}

	return nil
}

func (g *GameSeq) sequenceExploreBySingleCell(columns *[]ExploreAreaOut, area ExploreAreaOut) error {
	extractedAreaTreasures := 0

	for x := area.Area.PosX; x < area.Area.PosX+area.Area.SizeX; x++ {
		if extractedAreaTreasures == area.Amount {
			return nil
		} else if x+1 == area.Area.PosX+area.Area.SizeX {
			lastArea := ExploreAreaOut{
				Area:   ExploreAreaIn{PosX: x, PosY: area.Area.PosY, SizeX: 1, SizeY: area.Area.SizeY},
				Amount: area.Amount - extractedAreaTreasures,
			}

			atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)

			*columns = append(*columns, lastArea)
			return nil
		}

		column, err := g.client.PostExplore(x, area.Area.PosY, 1, 1)
		if err != nil {
			return err
		}

		g.gameStats.writeExploreCost(1)

		atomic.AddUint64(&g.gameStats.SingleCellExplores, 1)

		if column.Amount == 0 {
			continue
		}

		atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)

		*columns = append(*columns, column)
		extractedAreaTreasures += column.Amount
	}

	return nil
}

func (g *GameSeq) issueLicense() error {
	var license License
	var err error

	for {
		if len(g.coinsQueue) != 0 {
			/*g.gameStats.licenseWaitLock.Lock()
			if g.gameStats.licenseWait >= 90 && len(g.coinsQueue) >= 11 {
				coins := make([]Coin, 0, 11)

				for i := 0; i < 11; i++ {
					coins = append(coins, <-g.coinsQueue)
				}

				g.gameStats.licenseWait -= 90
				g.gameStats.licenseWaitLock.Unlock()

				license, err = g.client.PostLicenses(coins)

				atomic.AddUint64(&g.gameStats.PaidLicenses, 1)
				atomic.AddUint64(&g.gameStats.ExpensiveLicenses, 1)
			} else {
				g.gameStats.licenseWaitLock.Unlock()

				license, err = g.client.PostLicenses([]Coin{<-g.coinsQueue})
				atomic.AddUint64(&g.gameStats.PaidLicenses, 1)
			}*/

			license, err = g.client.PostLicenses([]Coin{<-g.coinsQueue})
			atomic.AddUint64(&g.gameStats.PaidLicenses, 1)
		} else {
			license, err = g.client.PostLicenses(nil)
			atomic.AddUint64(&g.gameStats.FreeLicenses, 1)
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
