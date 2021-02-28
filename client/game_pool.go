package client

import (
	"errors"
	"log"
	"sync"
	"time"
)

const coinsMax = 2000

// GamePool с пулами потоков
type GamePool struct {
	client         *Client
	areasQueue     chan ExploreAreaOut
	coinsQueue     chan Coin
	treasuresQueue chan []Treasure
}

func NewGamePool(client *Client) *GamePool {
	return &GamePool{
		client:         client,
		areasQueue:     make(chan ExploreAreaOut, 200),
		coinsQueue:     make(chan Coin, coinsMax),
		treasuresQueue: make(chan []Treasure, 2000),
	}
}

func (g *GamePool) explore(yStart, yEnd int) error {
	for y := yStart; y < yEnd; y++ {
		//count := 0

		for x := 0; x < 3498; x += 3 {
			area, err := g.client.PostExplore(x, y, 3, 1)
			if err != nil {
				return err
			}

			if area.Amount >= 1 {
				g.areasQueue <- area
				//count++
			}

			/*if count == 100 {
				log.Printf("X: %d, Y: %d", x, y)
				break
			}*/

			/*if area.Amount == 2 {
				count++
			}

			if count == 10 {
				log.Printf("X: %d, Y: %d", x, y)
				break
			}*/
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

			if len(g.coinsQueue) != coinsMax {
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

			if area.Amount == 0 {
				continue
			}

			columnDepth := 1
			columnTreasures := 0
			for columnTreasures < column.Amount {
				if license.IsEnd() {
					if len(g.coinsQueue) != 0 {
						license, err = g.client.PostLicenses(<-g.coinsQueue)
					} else {
						license, err = g.client.PostLicenses(0)
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
						continue
					}

					return err
				}

				g.treasuresQueue <- treasures

				columnTreasures += len(treasures)
				columnDepth++
				license.DigUsed++
			}

			extractedAreaTreasures += columnTreasures
		}
	}

	return nil
}

func (g *GamePool) Run() error {
	if err := g.waitBackend(); err != nil {
		return err
	}

	/*areas := make([]int, 140)

	log.Printf("EXPLORE START: %s\n", time.Now().Format("15:04:05.000000"))
	for x := 0; x < 3500; x++ {
		area, _ := g.client.PostExplore(x, 0, 3, 1)
		areas = append(areas, area.Amount)
	}
	log.Printf("EXPLORE END: %s\n", time.Now().Format("15:04:05.000000"))
	log.Printf("LEN AREAS: %d\n", len(areas))

	sort.SliceStable(areas, func(i, j int) bool {
		return areas[i] >= areas[j]
	})

	log.Printf("AREAS: %+v\n", areas)*/

	go func() {
		time.Sleep(time.Second * 598)
		g.printStats()
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

func (g *GamePool) printStats() {
	log.Printf("COUNT REQ: %d\n", g.client.countReq)

	log.Printf("LICENSE 500: %d\n", g.client.licenses500Cnt)
	log.Printf("LICENSE 400: %d\n", g.client.licenses400Cnt)

	log.Printf("EXPLORE 500: %d\n", g.client.explore500Cnt)
	log.Printf("EXPLORE 400: %d\n", g.client.explore400Cnt)

	log.Printf("DIG 500: %d\n", g.client.dig500Cnt)
	log.Printf("DIG 400: %d\n", g.client.dig400Cnt)

	log.Printf("CASH 500: %d\n", g.client.cash500Cnt)
	log.Printf("CASH 400: %d\n", g.client.cash400Cnt)
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
