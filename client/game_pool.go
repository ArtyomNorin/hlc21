package client

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const coinsMax = 300

type GameStats struct {
	FreeLicenses                    uint64
	PaidLicenses                    uint64
	SingleCellExplores              uint64
	SingleCellExploresWithTreasures uint64
	DigsDone                        uint64
	TreasuresFound                  uint64
	TreasuresExchanged              uint64
	TreasuresNotExchanged           uint64
}

// GamePool с пулами потоков
type GamePool struct {
	gameStats      GameStats
	client         *Client
	areasQueue     chan ExploreAreaOut
	coinsQueue     chan Coin
	treasuresQueue chan Treasure
}

func NewGamePool(client *Client) *GamePool {
	return &GamePool{
		gameStats:      GameStats{},
		client:         client,
		areasQueue:     make(chan ExploreAreaOut, 30),
		coinsQueue:     make(chan Coin, 600),
		treasuresQueue: make(chan Treasure, 30),
	}
}

func (g *GamePool) explore(yStart, yEnd, chunkSize int) error {
	for y := yStart; y < yEnd; y++ {
		for x := 0; x < 3500-chunkSize; x += chunkSize {
			area, err := g.client.PostExplore(x, y, chunkSize, 1)
			if err != nil {
				return err
			}

			if area.Amount < 1 {
				continue
			}

			extractedAreaTreasures := 0

			for xArea := area.Area.PosX; xArea < area.Area.PosX+area.Area.SizeX; xArea++ {
				if extractedAreaTreasures == area.Amount {
					break
				}

				column, err := g.client.PostExplore(xArea, area.Area.PosY, 1, 1)
				if err != nil {
					return err
				}

				atomic.AddUint64(&g.gameStats.SingleCellExplores, 1)

				if column.Amount == 0 {
					continue
				}

				atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)

				g.areasQueue <- column
				extractedAreaTreasures++
			}
		}
	}

	return nil
}

func (g *GamePool) binExploreRow(area ExploreAreaOut) error {
	leftArea, err := g.client.PostExplore(area.Area.PosX, area.Area.PosY, area.Area.SizeX/2, area.Area.SizeY)
	if err != nil {
		return err
	}

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

			return g.binExploreRow(rightArea)
		}

		// если сокровища и в левой и в правой части, то исследуем обе части
		if err := g.binExploreRow(leftArea); err != nil {
			return err
		}

		rightArea := leftArea
		rightArea.Area.PosX += leftArea.Area.SizeX
		rightArea.Amount = area.Amount - leftArea.Amount

		return g.binExploreRow(rightArea)
	}

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

	// т.к. в левой ячейке не все сокровища то правую ячейку тоже сохраняем
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

		if coins == nil {
			atomic.AddUint64(&g.gameStats.TreasuresNotExchanged, 1)
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
	license := License{}
	var lErr error

	for column := range g.areasQueue {
		columnDepth := 1
		columnTreasures := 0
		for columnTreasures < column.Amount {
			if license.IsEnd() {
				if len(g.coinsQueue) != 0 {
					license, lErr = g.client.PostLicenses(<-g.coinsQueue)
					atomic.AddUint64(&g.gameStats.PaidLicenses, 1)
				} else {
					license, lErr = g.client.PostLicenses(0)
					atomic.AddUint64(&g.gameStats.FreeLicenses, 1)
				}

				if lErr != nil {
					return lErr
				}
			}

			treasures, err := g.client.PostDig(license.ID, column.Area.PosX, column.Area.PosY, columnDepth)
			if err != nil {
				if errors.Is(err, NoTreasureErr) {
					columnDepth++
					license.DigUsed++
					atomic.AddUint64(&g.gameStats.DigsDone, 1)
					continue
				}

				return err
			}

			atomic.AddUint64(&g.gameStats.DigsDone, 1)

			for _, treasure := range treasures {
				g.treasuresQueue <- treasure
			}

			columnTreasures += len(treasures)
			columnDepth++
			license.DigUsed++

			atomic.AddUint64(&g.gameStats.TreasuresFound, uint64(len(treasures)))
		}
	}

	return nil
}

func (g *GamePool) Run() error {
	/*amounts := make([]int, 0)

	chunk := 128

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

	countExplorers := 6
	countDiggers := 10
	countCashiers := 8
	exploreChunk := 3

	fmt.Printf(
		"GamePool. SequenceExplore %dx1. %d explorers. %d diggers. %d cashiers. Rps limit 1000.\n",
		exploreChunk, countExplorers, countDiggers, countCashiers,
	)
	if err := g.waitBackend(); err != nil {
		return err
	}

	go func() {
		time.Sleep(time.Second * 598)
		g.printGameStats()
		g.printHttpStats()
	}()

	go func() {
		for {
			time.Sleep(time.Second)
			g.printQueueStats()
		}
	}()

	wg := new(sync.WaitGroup)

	// *** EXPLORE ***
	yChunkSize := 3500 / countExplorers

	for y := 0; y <= 3500-yChunkSize; y += yChunkSize {
		wg.Add(1)
		go func(yStart, yEnd int) {
			if err := g.explore(yStart, yEnd, exploreChunk); err != nil {
				log.Fatalln(err)
			}

			wg.Done()
		}(y, y+yChunkSize)
	}

	// *** DIG ***
	for i := 0; i < countDiggers; i++ {
		wg.Add(1)
		go func() {
			if err := g.dig(); err != nil {
				log.Fatalln(err)
			}

			wg.Done()
		}()
	}

	// *** CASH ***
	for i := 0; i < countCashiers; i++ {
		wg.Add(1)
		go func() {
			if err := g.cash(); err != nil {
				log.Fatalln(err)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	g.printGameStats()
	g.printHttpStats()

	return nil

	/*fmt.Println("GamePool. SimpleExplore 3x1. 10 explorers. Rps limit 1000.")

	go func() {
		time.Sleep(time.Second * 598)
		g.printGameStats()
		g.printHttpStats()
	}()

	wg := new(sync.WaitGroup)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			if err := g.simpleExplore(); err != nil {
				log.Fatalln(err)
			}

			wg.Done()
		}()
	}

	countThreads := 10
	yChunkSize := 3500 / countThreads

	for y := 0; y <= 3500-yChunkSize; y += yChunkSize {
		go func(yStart, yEnd int) {
			if err := g.simplePExplore(yStart, yEnd); err != nil {
				log.Fatalln(err)
			}
		}(y, y+yChunkSize)
	}

	wg.Wait()

	return nil*/
}

func (g *GamePool) printQueueStats() {
	log.Printf("AREAS QUEUE: %d\n", len(g.areasQueue))
	log.Printf("TREASURES QUEUE: %d\n", len(g.treasuresQueue))
	log.Printf("COINS QUEUE: %d\n", len(g.coinsQueue))
}

func (g *GamePool) printGameStats() {
	fmt.Println("*** GAME STATS ***")

	fmt.Printf("Free licenses: %d\n", g.gameStats.FreeLicenses)
	fmt.Printf("Paid licenses: %d\n", g.gameStats.PaidLicenses)

	fmt.Printf("Single cell explores: %d\n", g.gameStats.SingleCellExplores)
	fmt.Printf("Single cell explores with treasures: %d\n", g.gameStats.SingleCellExploresWithTreasures)

	fmt.Printf("Digs done: %d\n", g.gameStats.DigsDone)

	fmt.Printf("Treasures found: %d\n", g.gameStats.TreasuresFound)
	fmt.Printf("Treasures exchanged: %d\n", g.gameStats.TreasuresExchanged)
	fmt.Printf("Treasures not exchanged: %d\n", g.gameStats.TreasuresNotExchanged)
}

func (g *GamePool) printHttpStats() {
	fmt.Println("*** HTTP STATS ***")

	fmt.Printf("Requests count: %d\n", g.client.countReq)

	fmt.Printf("Licenses 500: %d\n", g.client.licenses500Cnt)
	fmt.Printf("Licenses 400: %d\n", g.client.licenses400Cnt)

	fmt.Printf("Explores 200: %d\n", g.client.explore200Cnt)
	fmt.Printf("Explores 500: %d\n", g.client.explore500Cnt)
	fmt.Printf("Explores 400: %d\n", g.client.explore400Cnt)

	fmt.Printf("Digs 500: %d\n", g.client.dig500Cnt)
	fmt.Printf("Digs 400: %d\n", g.client.dig400Cnt)

	fmt.Printf("Cashes 500: %d\n", g.client.cash500Cnt)
	fmt.Printf("Cashes 400: %d\n", g.client.cash400Cnt)
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
