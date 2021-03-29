package client

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type GamePoolSeq struct {
	gameStats           *GameStats
	client              *Client
	coinsQueue          chan Coin
	licenses            chan SyncLicense
	areasQueue          chan ExploreAreaOut
	treasuresQueue      chan Treasure
	coinsTreasuresQueue chan Treasure
	exploreStartSignal  chan int
	exploreFinishSignal chan struct{}
	digStartSignal      chan struct{}
	digFinishSignal     chan struct{}
	cashStartSignal     chan struct{}
	cashFinishSignal    chan struct{}
}

func NewGamePoolSeq(client *Client) *GamePoolSeq {
	return &GamePoolSeq{
		gameStats: &GameStats{
			exploreCostLock: new(sync.Mutex),
			digCostLock:     new(sync.Mutex),
			cashCostLock:    new(sync.Mutex),
			licenseWaitLock: new(sync.Mutex),
		},
		client:              client,
		coinsQueue:          make(chan Coin, 600),
		licenses:            make(chan SyncLicense, 100),
		areasQueue:          make(chan ExploreAreaOut, 500),
		treasuresQueue:      make(chan Treasure, 500),
		coinsTreasuresQueue: make(chan Treasure, 500),
		exploreStartSignal:  make(chan int, 20),
		exploreFinishSignal: make(chan struct{}, 20),
		digStartSignal:      make(chan struct{}, 20),
		digFinishSignal:     make(chan struct{}, 20),
		cashStartSignal:     make(chan struct{}, 20),
		cashFinishSignal:    make(chan struct{}, 20),
	}
}

func (g *GamePoolSeq) Run() error {
	exploreType := SequenceExploreType
	countExplorers := 4
	countDiggers := 4
	countCashiers := 4
	exploreChunk := 30 //28 bin 30 seq
	skipLevels := 2

	log.Printf(
		"GamePoolSeq. %s %dx1. %d explorers. %d dig. %d cash. %d lvls skip\n",
		exploreType, exploreChunk, countExplorers, countDiggers, countCashiers, skipLevels,
	)

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

	// *** COINS CASH ***
	for i := 0; i < 1; i++ {
		go func() {
			for treasure := range g.coinsTreasuresQueue {
				if err := g.coinsCash(treasure); err != nil {
					log.Fatalln(err)
				}
			}
		}()
	}

	// *** EXPLORE ***
	xChunk := 3500 / countExplorers

	for i := 0; i < countExplorers; i++ {
		go func(xStart, xEnd int) {
			for {
				y := <-g.exploreStartSignal

				switch exploreType {
				case BinExploreType:
					if err := g.binExplore(xStart, xEnd, y, exploreChunk); err != nil {
						log.Fatalln(err)
					}
				case SequenceExploreType:
					if err := g.sequenceExplore(xStart, xEnd, y, exploreChunk); err != nil {
						log.Fatalln(err)
					}
				}

				g.exploreFinishSignal <- struct{}{}
			}
		}(i*xChunk, i*xChunk+xChunk)
	}

	// *** DIG ***
	for i := 0; i < countDiggers; i++ {
		go func() {
			for {
				<-g.digStartSignal

			digLoop:
				for {
					select {
					case column := <-g.areasQueue:
						if err := g.dig(column, skipLevels); err != nil {
							log.Fatalln(err)
						}
					default:
						g.digFinishSignal <- struct{}{}
						break digLoop
					}
				}
			}
		}()
	}

	// *** CASH ***
	for i := 0; i < countCashiers; i++ {
		go func() {
			for {
				<-g.cashStartSignal

			cashLoop:
				for {
					select {
					case treasure := <-g.treasuresQueue:
						if err := g.cash(treasure); err != nil {
							log.Fatalln(err)
						}
					default:
						g.cashFinishSignal <- struct{}{}
						break cashLoop
					}
				}
			}
		}()
	}

	for y := 0; y < 3500; y++ {
		g.startExplorers(countExplorers, y)
		g.waitExplorers(countExplorers)

		g.startDiggers(countDiggers)
		g.waitDiggers(countDiggers)

		g.startCashiers(countCashiers)
		g.waitCashiers(countCashiers)
	}

	return nil
}

func (g *GamePoolSeq) startExplorers(countExplorers, y int) {
	for i := 0; i < countExplorers; i++ {
		g.exploreStartSignal <- y
	}
}

func (g *GamePoolSeq) waitExplorers(countExplorers int) {
	for i := 0; i < countExplorers; i++ {
		<-g.exploreFinishSignal
	}
}

func (g *GamePoolSeq) startDiggers(countDiggers int) {
	for i := 0; i < countDiggers; i++ {
		g.digStartSignal <- struct{}{}
	}
}

func (g *GamePoolSeq) waitDiggers(countDiggers int) {
	for i := 0; i < countDiggers; i++ {
		<-g.digFinishSignal
	}
}

func (g *GamePoolSeq) startCashiers(countCashiers int) {
	for i := 0; i < countCashiers; i++ {
		g.cashStartSignal <- struct{}{}
	}
}

func (g *GamePoolSeq) waitCashiers(countCashiers int) {
	for i := 0; i < countCashiers; i++ {
		<-g.cashFinishSignal
	}
}

func (g *GamePoolSeq) dig(column ExploreAreaOut, skipLevels int) error {
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
			if len(g.coinsQueue) <= coinsMax {
				for _, treasure := range treasures {
					g.coinsTreasuresQueue <- treasure
				}
			} else {
				for _, treasure := range treasures {
					g.treasuresQueue <- treasure
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

func (g *GamePoolSeq) cash(treasure Treasure) error {
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

	return nil
}

func (g *GamePoolSeq) coinsCash(treasure Treasure) error {
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

	if coins != nil {
		for _, coin := range coins {
			g.coinsQueue <- coin
		}
	}

	return nil
}

func (g *GamePoolSeq) binExplore(xStart, xEnd, y, xChunk int) error {
	for x := xStart; x < xEnd-xChunk; x += xChunk {
		area, err := g.client.PostExplore(x, y, xChunk, 1)
		if err != nil {
			return err
		}

		g.gameStats.writeExploreCost(xChunk)

		if area.Amount >= 1 {
			if err := g.binExploreRow(area); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *GamePoolSeq) binExploreRow(area ExploreAreaOut) error {
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
			return g.binExploreRow(leftArea)
		}

		// если все сокровища в правой части,  то исследуем только правую часть
		if leftArea.Amount == 0 {
			rightArea := area.GetRightArea(leftArea)
			return g.binExploreRow(rightArea)
		}

		// если сокровища и в левой и в правой части, то исследуем обе части
		if err := g.binExploreRow(leftArea); err != nil {
			return err
		}

		rightArea := area.GetRightArea(leftArea)
		return g.binExploreRow(rightArea)
	}

	atomic.AddUint64(&g.gameStats.SingleCellExplores, 1)

	// если ячейка 1x1

	// если все сокровища в левой ячейке, то сохраняем ячейку
	if leftArea.Amount == area.Amount {
		g.areasQueue <- leftArea
		atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)
		return nil
	}

	// если в левой ячейке есть сокровище, то сохраняем её
	if leftArea.Amount > 0 {
		g.areasQueue <- leftArea
		atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)
	}

	if area.Area.SizeX%2 != 0 {
		rightArea := area.GetRightArea(leftArea)
		return g.binExploreRow(rightArea)
	}

	rightArea := area.GetRightArea(leftArea)
	g.areasQueue <- rightArea
	atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)

	return nil
}

func (g *GamePoolSeq) issueLicense() error {
	var license License
	var err error

	for {
		if len(g.coinsQueue) != 0 {
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

func (g *GamePoolSeq) sequenceExploreRow(area ExploreAreaOut, chunkSize int) error {
	extractedAreaTreasures := 0

	for xArea := area.Area.PosX; xArea < area.Area.PosX+area.Area.SizeX; xArea += chunkSize {
		if extractedAreaTreasures == area.Amount {
			return nil
		} else if xArea+chunkSize == area.Area.PosX+area.Area.SizeX {
			lastArea := ExploreAreaOut{
				Area:   ExploreAreaIn{PosX: xArea, PosY: area.Area.PosY, SizeX: chunkSize, SizeY: area.Area.SizeY},
				Amount: area.Amount - extractedAreaTreasures,
			}

			return g.sequenceExploreBySingleCell(lastArea)
		}

		subArea, err := g.client.PostExplore(xArea, area.Area.PosY, chunkSize, 1)
		if err != nil {
			return err
		}

		g.gameStats.writeExploreCost(chunkSize)

		if subArea.Amount < 1 {
			continue
		}

		if err := g.sequenceExploreBySingleCell(subArea); err != nil {
			return err
		}

		extractedAreaTreasures += subArea.Amount
	}

	return nil
}

func (g *GamePoolSeq) sequenceExplore(xStart, xEnd, y, xChunk int) error {
	for x := xStart; x < xEnd-xChunk; x += xChunk {
		area, err := g.client.PostExplore(x, y, xChunk, 1)
		if err != nil {
			return err
		}

		g.gameStats.writeExploreCost(xChunk)

		if area.Amount < 1 {
			continue
		}

		if err := g.sequenceExploreRow(area, 3); err != nil {
			return err
		}
	}

	return nil
}

func (g *GamePoolSeq) sequenceExploreBySingleCell(area ExploreAreaOut) error {
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

			g.areasQueue <- lastArea
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

		g.areasQueue <- column
		extractedAreaTreasures += column.Amount
	}

	return nil
}
