package client

import (
	"errors"
	"log"
	"sort"
	"sync"
	"time"
)

type Game struct {
	countThreads int
	client       *Client

	areaSize  int
	areas     []ExploreAreaOut //области отсортированные по кол-ву сокровищ
	areasChan chan []ExploreAreaOut

	countActiveLicenses int //кол-во одновременно активных лицензий
	countProcessAreas   int //кол-во областей для раскопок. Должно быть кратно countActiveLicenses
}

func NewGame(client *Client) *Game {
	areaSize := 25
	countAreas := (3500 / areaSize) * (3500 / areaSize)

	return &Game{
		countThreads: 4,
		client:       client,

		areaSize:  areaSize,
		areas:     make([]ExploreAreaOut, 0, countAreas),
		areasChan: make(chan []ExploreAreaOut),

		countActiveLicenses: 10,
		countProcessAreas:   530,
	}
}

func (g *Game) explorePhase() {
	yStart := 0
	yChank := 3500 / g.countThreads

	wg := new(sync.WaitGroup)
	wg.Add(g.countThreads + 1)

	go func() {
		defer wg.Done()

		for i := 0; i < g.countThreads; i++ {
			g.areas = append(g.areas, <-g.areasChan...)
		}

		close(g.areasChan)
	}()

	for i := 0; i < g.countThreads; i++ {
		go g.explore(yStart, yStart+yChank, wg)
		yStart += yChank
	}

	wg.Wait()
}

func (g *Game) digPhase() {
	areas := g.areas[:g.countProcessAreas]
	areaChunk := g.countProcessAreas / g.countActiveLicenses

	wg := new(sync.WaitGroup)
	wg.Add(g.countActiveLicenses)

	for i := 0; i < g.countProcessAreas; i += areaChunk {
		go g.dig(areas[i:i+areaChunk], wg)
	}

	wg.Wait()
}

func (g *Game) dig(areas []ExploreAreaOut, wg *sync.WaitGroup) {
	defer wg.Done()

	license := License{}
	coinsBank := make([]Coin, 0, 5000)
	var coin Coin

	for _, area := range areas {
		areaTreasuresCnt := 0 // кол-во выкопанных сокровищ для области

		x := area.Area.PosX
		y := area.Area.PosY

	Loop:
		for y < area.Area.PosY+area.Area.SizeY {
			for x < area.Area.PosX+area.Area.SizeX {
				if areaTreasuresCnt == area.Amount {
					break Loop
				}

				columnArea, err := g.client.PostExplore(x, y, 1, 1)
				if err != nil {
					log.Fatalln(err)
				}

				if columnArea.Amount == 0 {
					x++
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
							log.Fatalln(err)
						}
					}

					treasures, err := g.client.PostDig(license.ID, x, y, columnDepth)
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

				areaTreasuresCnt += columnTreasuresCnt
				x++
			}

			x = area.Area.PosX
			y++
		}
	}
}

func (g *Game) explore(yStart, yEnd int, wg *sync.WaitGroup) {
	defer wg.Done()

	xAreas := 3500 / g.areaSize
	yAreas := (yEnd - yStart) / g.areaSize

	areas := make([]ExploreAreaOut, 0, xAreas*yAreas)

	x := 0
	y := yStart

	for y+g.areaSize <= yEnd {
		for x+g.areaSize <= 3500 {
			exploreAreaOut, err := g.client.PostExplore(x, y, g.areaSize, g.areaSize)
			if err != nil {
				log.Fatalln(err)
			}

			x += g.areaSize
			areas = append(areas, exploreAreaOut)
		}

		x = 0
		y += g.areaSize
	}

	g.areasChan <- areas
}

func (g *Game) Run() error {
	if err := g.waitBackend(); err != nil {
		return err
	}

	log.Printf("EXPLORE START: %s\n", time.Now().Format("15:04:05.000000"))
	g.explorePhase()
	log.Printf("EXPLORE END: %s\n", time.Now().Format("15:04:05.000000"))
	exploreCountReq := g.client.countReq
	log.Printf("EXPLORE COUNT REQ: %d\n\n", exploreCountReq)

	log.Printf("AREAS LEN: %d\n\n", len(g.areas))

	log.Printf("SORT START: %s\n", time.Now().Format("15:04:05.000000"))
	sort.SliceStable(g.areas, func(i, j int) bool { return g.areas[i].Amount > g.areas[j].Amount })
	log.Printf("SORT END: %s\n\n", time.Now().Format("15:04:05.000000"))

	log.Printf("DIG START: %s\n", time.Now().Format("15:04:05.000000"))
	g.digPhase()
	log.Printf("DIG END: %s\n", time.Now().Format("15:04:05.000000"))
	log.Printf("DIG COUNT REQ: %d\n\n", g.client.countReq-exploreCountReq)

	//stats
	log.Printf("COUNT REQ: %d\n", g.client.countReq)

	log.Printf("LICENSE 500: %d\n", g.client.licenses500Cnt)
	log.Printf("LICENSE 400: %d\n", g.client.licenses400Cnt)

	log.Printf("EXPLORE 500: %d\n", g.client.explore500Cnt)
	log.Printf("EXPLORE 400: %d\n", g.client.explore400Cnt)

	log.Printf("DIG 500: %d\n", g.client.dig500Cnt)
	log.Printf("DIG 400: %d\n", g.client.dig400Cnt)

	log.Printf("CASH 500: %d\n", g.client.cash500Cnt)
	log.Printf("CASH 400: %d\n", g.client.cash400Cnt)

	return nil
}

func (g *Game) waitBackend() error {
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
