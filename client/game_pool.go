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

const coinsMax = 300
const BinExploreType = "BinExplore"
const SequenceExploreType = "SequenceExplore"
const RandomSequenceExploreType = "RandomSequenceExplore"
const RandomBinExploreType = "RandomBinExplore"

// GamePool с пулами потоков
type GamePool struct {
	gameStats      *GameStats
	client         *Client
	areasQueue     chan ExploreAreaOut
	coinsQueue     chan Coin
	treasuresQueue chan Treasure
	licenses       chan SyncLicense
}

func NewGamePool(client *Client) *GamePool {
	return &GamePool{
		gameStats: &GameStats{
			exploreCostLock: new(sync.Mutex),
			digCostLock:     new(sync.Mutex),
			cashCostLock:    new(sync.Mutex),
			licenseWaitLock: new(sync.Mutex),
		},
		client:         client,
		areasQueue:     make(chan ExploreAreaOut, 9),
		coinsQueue:     make(chan Coin, 600),
		treasuresQueue: make(chan Treasure, 10),
		licenses:       make(chan SyncLicense, 100),
	}
}

func (g *GamePool) issueLicense() error {
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

func (g *GamePool) sequenceExploreBySingleCell(area ExploreAreaOut) error {
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

func (g *GamePool) randomBinExplore(yStart, yEnd, chunkSize, reqPerX int) error {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomAreaSize := 3500 / reqPerX

	for y := yStart; y < yEnd; y++ {
		for i := 0; i < reqPerX; i++ {
			min := i * randomAreaSize
			max := min + randomAreaSize - chunkSize

			x := randomizer.Intn(max-min) + min

			area, err := g.client.PostExplore(x, y, chunkSize, 1)
			if err != nil {
				return err
			}

			g.gameStats.writeExploreCost(chunkSize)

			if area.Amount < 1 {
				continue
			}

			if err := g.binExploreRow(area); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *GamePool) randomSequenceExplore(yStart, yEnd, chunkSize, reqPerX int, treasuresLimit uint64) error {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomAreaSize := 3500 / reqPerX

	for y := yStart; y < yEnd; y++ {
		for i := 0; i < reqPerX; i++ {
			min := i * randomAreaSize
			max := min + randomAreaSize - chunkSize

			x := randomizer.Intn(max-min) + min

			area, err := g.client.PostExplore(x, y, chunkSize, 1)
			if err != nil {
				return err
			}

			g.gameStats.writeExploreCost(chunkSize)

			if area.Amount < 1 {
				continue
			}

			if err := g.sequenceExploreRow(area, 3); err != nil {
				return err
			}

			if g.gameStats.SingleCellExploresWithTreasures >= treasuresLimit {
				return nil
			}
		}
	}

	return nil
}

func (g *GamePool) sequenceExploreRow(area ExploreAreaOut, chunkSize int) error {
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

func (g *GamePool) sequenceExplore(yStart, yEnd, chunkSize int, treasuresLimit uint64) error {
	for y := yStart; y < yEnd; y++ {
		for x := 0; x < 3500-chunkSize; x += chunkSize {
			area, err := g.client.PostExplore(x, y, chunkSize, 1)
			if err != nil {
				return err
			}

			g.gameStats.writeExploreCost(chunkSize)

			if area.Amount < 1 {
				continue
			}

			if err := g.sequenceExploreRow(area, 3); err != nil {
				return err
			}

			if g.gameStats.SingleCellExploresWithTreasures >= treasuresLimit {
				return nil
			}
		}
	}

	return nil
}

func (g *GamePool) binExplore(yStart, yEnd, chunkSize int, treasuresLimit uint64) error {
	for y := yStart; y < yEnd; y++ {
		for x := 0; x < 3500-chunkSize; x += chunkSize {
			area, err := g.client.PostExplore(x, y, chunkSize, 1)
			if err != nil {
				return err
			}

			g.gameStats.writeExploreCost(chunkSize)

			if area.Amount >= 1 {
				if err := g.binExploreRow(area); err != nil {
					return err
				}
			}

			if g.gameStats.SingleCellExploresWithTreasures >= treasuresLimit {
				return nil
			}
		}

	}

	return nil
}

func (g *GamePool) binExploreRow(area ExploreAreaOut) error {
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
			rightArea := leftArea
			rightArea.Area.PosX += leftArea.Area.SizeX
			rightArea.Amount = area.Amount
			rightArea.Area.SizeX = area.Area.SizeX - leftArea.Area.SizeX

			return g.binExploreRow(rightArea)
		}

		// если сокровища и в левой и в правой части, то исследуем обе части
		if err := g.binExploreRow(leftArea); err != nil {
			return err
		}

		rightArea := leftArea
		rightArea.Area.PosX += leftArea.Area.SizeX
		rightArea.Amount = area.Amount - leftArea.Amount
		rightArea.Area.SizeX = area.Area.SizeX - leftArea.Area.SizeX

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
		rightArea := leftArea
		rightArea.Area.PosX += leftArea.Area.SizeX
		rightArea.Amount = area.Amount - leftArea.Amount
		rightArea.Area.SizeX = area.Area.SizeX - leftArea.Area.SizeX

		return g.binExploreRow(rightArea)
	}

	rightArea := leftArea
	rightArea.Area.PosX += leftArea.Area.SizeX
	rightArea.Amount = area.Amount - leftArea.Amount

	g.areasQueue <- rightArea
	atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)

	return nil
}

func (g *GamePool) cash() error {
	for treasure := range g.treasuresQueue {
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
	}

	return nil
}

func (g *GamePool) dig() error {
	for column := range g.areasQueue {
		columnDepth := 1
		columnTreasures := 0
		for columnTreasures < column.Amount {
			timeStart := time.Now()
			syncLicense := <-g.licenses
			g.gameStats.writeLicenseWait(time.Since(timeStart))

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

			if columnDepth > 2 {
				for _, treasure := range treasures {
					g.treasuresQueue <- treasure
				}
			}

			columnTreasures += len(treasures)
			columnDepth++
			syncLicense.DecreaseDig()

			atomic.AddUint64(&g.gameStats.TreasuresFound, uint64(len(treasures)))
		}
	}

	return nil
}

func (g *GamePool) printArea(area ExploreAreaOut) error {
	for x := area.Area.PosX; x < area.Area.PosX+area.Area.SizeX; x++ {
		cell, err := g.client.PostExplore(x, area.Area.PosY, 1, 1)
		if err != nil {
			return err
		}

		if cell.Amount == 0 {
			fmt.Print(0, " ")
		} else {
			fmt.Print(cell.Amount, " ")
		}
	}

	fmt.Println()
	return nil
}

func (g *GamePool) Run() error {
	/*fmt.Println("BinExplore 30x1. 8 explorers. Rps limit 1000.")

	go func() {
		time.Sleep(time.Second * 598)
		g.printGameStats()
		g.printHttpStats()
	}()

	wg := new(sync.WaitGroup)

	yChunkSize := 3500 / 8

	for y := 0; y <= 3500-yChunkSize; y += yChunkSize {
		wg.Add(1)
		go func(yStart, yEnd int) {
			if err := g.binExplore(yStart, yEnd, 30); err != nil {
				log.Fatalln(err)
			}

			wg.Done()
		}(y, y+yChunkSize)
	}

	wg.Wait()

	g.printGameStats()
	g.printHttpStats()

	return nil*/

	/*g.binExplore(0, 10, 28)

	g.printGameStats()
	g.printHttpStats()

	return nil*/
	/*chunk := 3
	fmt.Printf("SequenceExplore %dx1. 8 explorers. Rps limit 1000.", chunk)

	go func() {
		time.Sleep(time.Second * 598)
		g.printGameStats()
		g.printHttpStats()
	}()

	wg := new(sync.WaitGroup)

	yChunkSize := 3500 / 8

	for y := 0; y <= 3500-yChunkSize; y += yChunkSize {
		wg.Add(1)
		go func(yStart, yEnd int) {
			for yArea := yStart; yArea < yEnd; yArea++ {
				for x := 0; x < 3500-chunk; x += chunk {
					_, err := g.client.PostExplore(x, yArea, chunk, 1)
					if err != nil {
						log.Fatalln(err)
					}
				}
			}

			wg.Done()
		}(y, y+yChunkSize)
	}

	wg.Wait()

	g.printGameStats()
	g.printHttpStats()

	return nil*/

	/*amounts := make([]int, 0)

	chunk := 60

	for x := 0; x < 3500-chunk; x += chunk {
		area, err := g.client.PostExplore(x, 0, chunk, 1)
		if err != nil {
			return err
		}

		amounts = append(amounts, area.Amount)
	}

	sort.SliceStable(amounts, func(i, j int) bool {
		return amounts[i] > amounts[j]
	})

	fmt.Println(amounts)

	return nil*/

	/*go func() {
		time.Sleep(time.Second * 598)
		g.printGameStats()
		g.printHttpStats()
	}()

	timeStart := time.Now()

	for y := 0; y < 30; y++ {
		for x := 0; x < 3500-128; x += 128 {
			area, err := g.client.PostExplore(x, y, 128, 1)
			if err != nil {
				return err
			}

			if area.Amount >= 6 {
				if err := g.binExploreRow(area); err != nil {
					return err
				}
			}
		}
	}

	fmt.Println("*** BinExplore 128x1 area.Amount >= 6 ***")
	fmt.Println("TIME:", time.Since(timeStart).Seconds())
	fmt.Println("REQ CNT:", g.client.countReq)
	fmt.Println("SINGE CELL TREASURE:", g.gameStats.SingleCellExploresWithTreasures)
	fmt.Println("SINGE CELL TREASURE PER SEC:", float64(g.gameStats.SingleCellExploresWithTreasures)/time.Since(timeStart).Seconds())
	fmt.Println("REQ PER SINGE CELL TREASURE:", float64(g.client.countReq)/float64(g.gameStats.SingleCellExploresWithTreasures))
	return nil*/

	exploreType := SequenceExploreType
	countExplorers := 10
	countDiggers := 9
	countCashiers := 10
	exploreChunk := 30
	treasuresLimit := 34000

	fmt.Printf(
		"%s %dx1. %d explorers. %d diggers. %d cashiers. %d treasures limit. Rps limit 1000.\n",
		exploreType, exploreChunk, countExplorers, countDiggers, countCashiers, treasuresLimit,
	)
	if err := g.waitBackend(); err != nil {
		return err
	}

	go func() {
		time.Sleep(time.Second * 598)
		g.gameStats.printStats()
		fmt.Println("----------------------")
		g.client.printStats()
	}()

	go func() {
		for {
			time.Sleep(time.Second)
			g.printQueueStats()
		}
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

	// *** EXPLORE ***
	exploreStartTime := time.Now()
	exploresDoneChan := make(chan struct{}, countExplorers)

	go func() {
		for i := 0; i < countExplorers; i++ {
			<-exploresDoneChan
		}

		log.Printf("Explore finish: %f\n", time.Since(exploreStartTime).Seconds())
		close(g.areasQueue)
	}()

	yChunkSize := 3500 / countExplorers

	for y := 0; y <= 3500-yChunkSize; y += yChunkSize {
		wg.Add(1)
		go func(yStart, yEnd int) {
			switch exploreType {
			case SequenceExploreType:
				if err := g.sequenceExplore(yStart, yEnd, exploreChunk, uint64(treasuresLimit)); err != nil {
					log.Fatalln(err)
				}
			case BinExploreType:
				if err := g.binExplore(yStart, yEnd, exploreChunk, uint64(treasuresLimit)); err != nil {
					log.Fatalln(err)
				}
			case RandomSequenceExploreType:
				if err := g.randomSequenceExplore(yStart, yEnd, exploreChunk, 100, uint64(treasuresLimit)); err != nil {
					log.Fatalln(err)
				}
			case RandomBinExploreType:
				if err := g.randomBinExplore(yStart, yEnd, exploreChunk, 100); err != nil {
					log.Fatalln(err)
				}
			}

			wg.Done()

			exploresDoneChan <- struct{}{}
		}(y, y+yChunkSize)
	}

	// *** DIG ***
	digStartTime := time.Now()
	digDoneChan := make(chan struct{}, countDiggers)

	go func() {
		for i := 0; i < countDiggers; i++ {
			<-digDoneChan
		}

		log.Printf("Dig finish: %f\n", time.Since(digStartTime).Seconds())
		close(g.treasuresQueue)
	}()

	for i := 0; i < countDiggers; i++ {
		wg.Add(1)
		go func() {
			if err := g.dig(); err != nil {
				log.Fatalln(err)
			}

			wg.Done()
			digDoneChan <- struct{}{}
		}()
	}

	// *** CASH ***
	cashStartTime := time.Now()
	cashDoneChan := make(chan struct{}, countCashiers)

	go func() {
		for i := 0; i < countCashiers; i++ {
			<-cashDoneChan
		}

		log.Printf("Cash finish: %f\n", time.Since(cashStartTime).Seconds())
	}()

	for i := 0; i < countCashiers; i++ {
		wg.Add(1)
		go func() {
			if err := g.cash(); err != nil {
				log.Fatalln(err)
			}
			wg.Done()
			cashDoneChan <- struct{}{}
		}()
	}

	wg.Wait()

	g.gameStats.printStats()
	fmt.Println("----------------------")
	g.client.printStats()

	return nil
}

func (g *GamePool) printQueueStats() {
	log.Printf("AREAS QUEUE: %d\n", len(g.areasQueue))
	log.Printf("TREASURES QUEUE: %d\n", len(g.treasuresQueue))
	log.Printf("COINS QUEUE: %d\n", len(g.coinsQueue))
}

func (g *GamePool) waitBackend() error {
	for {
		isBackendAlive, err := g.client.HealthCheck()
		if err != nil {
			return err
		}

		if isBackendAlive {
			log.Println("backend is alive")
			return nil
		}

		time.Sleep(time.Millisecond * 50)
		log.Println("wait backend")
	}
}
