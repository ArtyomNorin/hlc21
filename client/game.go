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
		countProcessAreas:   300,
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

	for _, area := range areas {
		areaTreasuresCnt := 0 // кол-во выкопанных сокровищ

		x := area.Area.PosX
		y := area.Area.PosY

	Loop:
		for y < area.Area.PosY+area.Area.SizeY {
			for x < area.Area.PosX+area.Area.SizeX {
				if area.Amount != 0 && areaTreasuresCnt == area.Amount {
					break Loop
				}

				columnTreasuresCnt := 0
				columnArea, err := g.client.PostExplore(x, y, 1, 1)
				if err != nil {
					if errors.Is(err, TooManyRequestsErr) || errors.Is(err, InternalServerErr) {
						log.Println(err)
						time.Sleep(time.Millisecond * 200)
						continue
					}

					log.Fatalln(err)
				}

				if columnArea.Amount == 0 {
					x++
					continue
				}

				columnDepth := 1
				for columnTreasuresCnt < columnArea.Amount {
					if license.IsEnd() {
						license, err = g.client.PostLicenses()
						if err != nil {
							log.Fatalln(err)
						}
					}

					treasures, err := g.client.PostDig(license.ID, x, y, columnDepth)
					if err != nil {
						if errors.Is(err, TooManyRequestsErr) || errors.Is(err, InternalServerErr) {
							log.Println(err)
							time.Sleep(time.Millisecond * 200)
							continue
						}

						if errors.Is(err, NoTreasureErr) {
							columnDepth++
							license.DigUsed++
							continue
						}

						log.Fatalln(err)
					}

					if len(treasures) == 0 {
						columnDepth++
						license.DigUsed++
						continue
					}

					for _, treasure := range treasures {
						if err := g.client.PostCash(treasure); err != nil {
							log.Fatalln(err)
						}

						columnTreasuresCnt++
						areaTreasuresCnt++
					}

					columnDepth++
					license.DigUsed++
				}

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
				if errors.Is(err, TooManyRequestsErr) || errors.Is(err, InternalServerErr) {
					log.Println(err)
					time.Sleep(time.Millisecond * 200)
					continue
				}

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

	timeStart := time.Now()
	g.explorePhase()
	totalTime := time.Since(timeStart)

	log.Printf("EXPLORE TIME: %s\n", totalTime.String())
	log.Printf("AREAS LEN: %d\n", len(g.areas))

	sort.SliceStable(g.areas, func(i, j int) bool {
		return g.areas[i].Amount > g.areas[j].Amount
	})

	timeStart = time.Now()
	g.digPhase()
	totalTime = time.Since(timeStart)

	log.Printf("DIG TIME: %s\n", totalTime.String())

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

		time.Sleep(time.Millisecond * 200)
		log.Println("wait backend")
	}
}
