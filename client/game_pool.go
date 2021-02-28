package client

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const coinsMax = 2000

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
	treasuresQueue chan []Treasure
}

func NewGamePool(client *Client) *GamePool {
	return &GamePool{
		gameStats:      GameStats{},
		client:         client,
		areasQueue:     make(chan ExploreAreaOut, 200),
		coinsQueue:     make(chan Coin, coinsMax),
		treasuresQueue: make(chan []Treasure, 2000),
	}
}

func (g *GamePool) explore(yStart, yEnd int) error {
	for y := yStart; y < yEnd; y++ {
		for x := 0; x < 3498; x += 3 {
			area, err := g.client.PostExplore(x, y, 3, 1)
			if err != nil {
				return err
			}

			if area.Amount >= 1 {
				g.areasQueue <- area
			}
		}
	}

	return nil
}

func (g *GamePool) cash() error {
	for treasures := range g.treasuresQueue {
		for _, treasure := range treasures {
			coins, err := g.client.PostCash(treasure)
			if err != nil {
				return err
			}

			if coins == nil {
				atomic.AddUint64(&g.gameStats.TreasuresNotExchanged, 1)
			} else {
				atomic.AddUint64(&g.gameStats.TreasuresExchanged, 1)
			}

			if coins != nil && len(g.coinsQueue) != coinsMax {
				for _, coin := range coins {
					g.coinsQueue <- coin
				}
			}
		}
	}

	return nil
}

func (g *GamePool) dig() error {
	license := License{}

	for area := range g.areasQueue {
		extractedAreaTreasures := 0

		for x := area.Area.PosX; x < area.Area.PosX+area.Area.SizeX; x++ {
			if extractedAreaTreasures == area.Amount {
				break
			}

			column, err := g.client.PostExplore(x, area.Area.PosY, 1, 1)
			if err != nil {
				return err
			}

			atomic.AddUint64(&g.gameStats.SingleCellExplores, 1)

			if column.Amount == 0 {
				continue
			}

			atomic.AddUint64(&g.gameStats.SingleCellExploresWithTreasures, 1)

			columnDepth := 1
			columnTreasures := 0
			for columnTreasures < column.Amount {
				if license.IsEnd() {
					if len(g.coinsQueue) != 0 {
						license, err = g.client.PostLicenses(<-g.coinsQueue)
						atomic.AddUint64(&g.gameStats.PaidLicenses, 1)
					} else {
						license, err = g.client.PostLicenses(0)
						atomic.AddUint64(&g.gameStats.FreeLicenses, 1)
					}

					if err != nil {
						return err
					}
				}

				treasures, err := g.client.PostDig(license.ID, x, area.Area.PosY, columnDepth)
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

				g.treasuresQueue <- treasures

				columnTreasures += len(treasures)
				columnDepth++
				license.DigUsed++
			}

			extractedAreaTreasures += columnTreasures
		}

		atomic.AddUint64(&g.gameStats.TreasuresFound, uint64(extractedAreaTreasures))
	}

	return nil
}

func (g *GamePool) Run() error {
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

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			if err := g.cash(); err != nil {
				log.Fatalln(err)
			}
			wg.Done()
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			if err := g.dig(); err != nil {
				log.Fatalln(err)
			}

			wg.Done()
		}()
	}

	countThreads := 10
	yChunkSize := 3500 / countThreads

	for y := 0; y <= 3500-yChunkSize; y += yChunkSize {
		go func(yStart, yEnd int) {
			if err := g.explore(yStart, yEnd); err != nil {
				log.Fatalln(err)
			}
		}(y, y+yChunkSize)
	}

	wg.Wait()

	return nil
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
