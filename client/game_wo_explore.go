package client

import (
	"errors"
	"log"
	"sync"
	"time"
)

// GameWOExplore без фазы explore
type GameWOExplore struct {
	countThreads int
	client       *Client
}

func NewGameWOExplore(client *Client) *GameWOExplore {
	return &GameWOExplore{
		countThreads: 10,
		client:       client,
	}
}

func (g *GameWOExplore) Run() error {
	if err := g.waitBackend(); err != nil {
		return err
	}

	go func() {
		time.Sleep(time.Second*598)
		g.printStats()
	}()

	wg := new(sync.WaitGroup)
	wg.Add(g.countThreads)

	threadChunkSize := 3500 / g.countThreads

	for y := 0; y <= 3500-threadChunkSize; y += threadChunkSize {
		go g.exploreAndDig(y, y+threadChunkSize, 5, 2, wg)
	}

	wg.Wait()

	return nil
}

func (g *GameWOExplore) exploreAndDig(yStart, yEnd, chunkSize, treasuresMin int, wg *sync.WaitGroup) {
	defer wg.Done()

	license := License{}
	coinsBank := make([]Coin, 0, 5000)
	var coin Coin

	for y := yStart; y <= yEnd-chunkSize; y += chunkSize {
		for x := 0; x <= 3500-chunkSize; x += chunkSize {
			chunkArea, err := g.client.PostExplore(x, y, chunkSize, chunkSize)
			if err != nil {
				log.Fatal(err)
			}

			if chunkArea.Amount < treasuresMin {
				continue
			}

			chunkTreasuresCnt := 0
			xChunk, yChunk := chunkArea.Area.PosX, chunkArea.Area.PosY

		loop:
			for yChunk < chunkArea.Area.PosY+chunkArea.Area.SizeY {
				for xChunk < chunkArea.Area.PosX+chunkArea.Area.SizeX {
					if chunkTreasuresCnt == chunkArea.Amount {
						break loop
					}

					columnArea, err := g.client.PostExplore(xChunk, yChunk, 1, 1)
					if err != nil {
						log.Fatal(err)
					}

					if columnArea.Amount == 0 {
						xChunk++
						continue
					}

					columnDepth := 1
					columnTreasuresCnt := 0

					for columnTreasuresCnt < columnArea.Amount {
						if license.IsEnd() {
							if len(coinsBank) != 0 {
								coin, coinsBank = coinsBank[len(coinsBank)-1], coinsBank[:len(coinsBank)-1]
								license, err = g.client.PostLicenses(coin)
							} else {
								license, err = g.client.PostLicenses(0)
							}

							if err != nil {
								log.Fatal(err)
							}
						}

						treasures, err := g.client.PostDig(license.ID, xChunk, yChunk, columnDepth)
						if err != nil {
							if errors.Is(err, NoTreasureErr) {
								columnDepth++
								license.DigUsed++
								continue
							}

							log.Fatalln(err)
						}

						for _, treasure := range treasures {
							coins, err := g.client.PostCash(treasure)
							if err != nil {
								log.Fatalln(err)
							}

							if len(coins) != 0 && len(coinsBank) != cap(coinsBank) {
								coinsBank = append(coinsBank, coins...)
							}
						}

						columnTreasuresCnt += len(treasures)
						columnDepth++
						license.DigUsed++
					}

					chunkTreasuresCnt += columnTreasuresCnt
					xChunk++
				}

				xChunk = chunkArea.Area.PosX
				yChunk++
			}
		}
	}
}

func (g *GameWOExplore) printStats() {
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

func (g *GameWOExplore) waitBackend() error {
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
