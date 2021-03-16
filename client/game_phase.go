package client

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// GamePhase разделение на фазу поиска и раскопок + продажи
type GamePhase struct {
	gameStats GameStats
	client    *Client
}

func NewGamePhase(client *Client) *GamePhase {
	return &GamePhase{
		gameStats: GameStats{},
		client:    client,
	}
}

func (g *GamePhase) Run() error {
	fmt.Println("GamePhase. BinExploreRowNew chunk 4x1, 10 workers. Rps limit 1000.")

	/*area, err := g.client.PostExplore(664, 0, 8, 8)
	if err != nil {
		return err
	}

	fmt.Println(area)
	return nil*/

	/*amounts := make([]int, 0)

	for x := 0; x < 3500-32; x+=32 {
		area, err := g.client.PostExplore(x, 0, 32, 1)
		if err != nil {
			return err
		}

		if area.Amount >= 1 {
			amounts = append(amounts, area.Amount)
		}
	}

	sort.SliceStable(amounts, func(i, j int) bool {
		return amounts[i] > amounts[j]
	})

	fmt.Println(amounts)
	return nil*/

	/*area, err := g.client.PostExplore(275, 704, 1, 1)
	if err != nil {
		return err
	}

	fmt.Println(area)
	for x := 0; x < area.Area.SizeX; x++ {
		for y := 0; y < area.Area.SizeY; y++ {
			area, err := g.client.PostExplore(area.Area.PosX+x, area.Area.PosY+y, 1, 1)
			if err != nil {
				return err
			}

			fmt.Print(area.Amount, " ")
		}
	}

	worker := NewPhaseWorker(g, 16)
	if err := worker.binExploreRow(area); err != nil {
		return err
	}

	g.printGameStats()
	g.printHttpStats()
	return nil*/

	/*worker := NewPhaseWorker(g, 16)

	timeStart := time.Now()

	for y := 0; y < 10; y++ {
		for x := 0; x < 3500-16; x += 16 {
			area, err := g.client.PostExplore(x, y, 16, 1)
			if err != nil {
				return err
			}

			if area.Amount >= 1 {
				if err := worker.binExploreRow(area); err != nil {
					return err
				}
			}
		}
	}

	fmt.Println("*** BinExplore 64x1 area.Amount > 0 ***")
	fmt.Println("TIME:", time.Since(timeStart).Seconds())
	fmt.Println("REQ CNT:", g.client.countReq)
	fmt.Println("SINGE CELL TREASURE:", g.gameStats.SingleCellExploresWithTreasures)
	fmt.Println("SINGE CELL TREASURE PER SEC:", float64(g.gameStats.SingleCellExploresWithTreasures)/time.Since(timeStart).Seconds())
	fmt.Println("REQ PER SINGE CELL TREASURE:", float64(g.client.countReq)/float64(g.gameStats.SingleCellExploresWithTreasures))
	return nil*/

	go func() {
		time.Sleep(time.Second * 598)
		g.printGameStats()
		g.printHttpStats()
	}()

	wg := new(sync.WaitGroup)

	countWorkers := 5
	treasuresTotal := 300000

	chunkSize := 3500 / countWorkers
	treasuresPerWorker := treasuresTotal / countWorkers

	for y := 0; y <= 3500-chunkSize; y += chunkSize {
		wg.Add(1)
		go func(yStart, yEnd int) {
			phaseWorker := NewPhaseWorker(g, treasuresPerWorker)
			if err := phaseWorker.Run(yStart, yEnd, 16); err != nil {
				log.Fatalln(err)
			}
			wg.Done()
		}(y, y+chunkSize)
	}

	wg.Wait()

	log.Println("GAME FINISH")

	return nil
}

func (g *GamePhase) printGameStats() {
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

func (g *GamePhase) printHttpStats() {
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

type PhaseWorker struct {
	game         *GamePhase
	digAreasSize int
	digAreas     []ExploreAreaOut
	coins        []Coin
	treasures    []Treasure
}

func NewPhaseWorker(game *GamePhase, areasSize int) *PhaseWorker {
	return &PhaseWorker{
		game:         game,
		digAreasSize: areasSize,
		digAreas:     make([]ExploreAreaOut, 0, areasSize),
		coins:        make([]Coin, 0, 200),
		treasures:    make([]Treasure, 0, 20000),
	}
}

func (pw *PhaseWorker) Run(yStart, yEnd, chunkSize int) error {
	log.Println("START EXPLORE PHASE")
	if err := pw.explore(yStart, yEnd, chunkSize); err != nil {
		return err
	}

	/*log.Println("START DIG PHASE")
	if err := pw.dig(); err != nil {
		return err
	}

	log.Println("START CASH PHASE")
	if err := pw.cash(); err != nil {
		return err
	}*/

	return nil
}

func (pw *PhaseWorker) cash() error {
	for _, treasure := range pw.treasures {
		coins, err := pw.game.client.PostCash(treasure)
		if err != nil {
			return err
		}

		if coins == nil {
			atomic.AddUint64(&pw.game.gameStats.TreasuresNotExchanged, 1)
		} else {
			atomic.AddUint64(&pw.game.gameStats.TreasuresExchanged, 1)
		}
	}

	return nil
}

func (pw *PhaseWorker) dig() error {
	var coin Coin
	var lErr error
	license := License{}

	for _, column := range pw.digAreas {
		columnDepth := 1
		columnTreasures := 0

		for columnTreasures < column.Amount {
			if license.IsEnd() {
				if len(pw.coins) != 0 {
					coin, pw.coins = pw.coins[0], pw.coins[1:]

					license, lErr = pw.game.client.PostLicenses(coin)
					atomic.AddUint64(&pw.game.gameStats.PaidLicenses, 1)
				} else {
					license, lErr = pw.game.client.PostLicenses(0)
					atomic.AddUint64(&pw.game.gameStats.FreeLicenses, 1)
				}

				if lErr != nil {
					return lErr
				}
			}

			treasures, err := pw.game.client.PostDig(license.ID, column.Area.PosX, column.Area.PosY, columnDepth)
			if err != nil {
				if errors.Is(err, NoTreasureErr) {
					columnDepth++
					license.DigUsed++
					atomic.AddUint64(&pw.game.gameStats.DigsDone, 1)
					continue
				}

				return err
			}

			atomic.AddUint64(&pw.game.gameStats.DigsDone, 1)

			columnTreasures += len(treasures)
			columnDepth++
			license.DigUsed++

			if len(pw.coins) < 100 {
				for _, treasure := range treasures {
					coins, err := pw.game.client.PostCash(treasure)
					if err != nil {
						return err
					}

					if coins == nil {
						atomic.AddUint64(&pw.game.gameStats.TreasuresNotExchanged, 1)
					} else {
						pw.coins = append(pw.coins, coins...)
						atomic.AddUint64(&pw.game.gameStats.TreasuresExchanged, 1)
					}
				}
			} else {
				pw.treasures = append(pw.treasures, treasures...)
			}

			atomic.AddUint64(&pw.game.gameStats.TreasuresFound, uint64(len(treasures)))
		}
	}

	return nil
}

func (pw *PhaseWorker) explore(yStart, yEnd, chunkSize int) error {
	for y := yStart; y < yEnd; y++ {
		for x := 0; x < 3500-chunkSize; x += chunkSize {
			area, err := pw.game.client.PostExplore(x, y, chunkSize, 1)
			if err != nil {
				return err
			}

			if area.Amount >= 1 {
				if err := pw.binExploreRow(area); err != nil {
					return err
				}

				if len(pw.digAreas) >= pw.digAreasSize {
					return nil
				}
			}
		}
	}

	return nil
}

func (pw *PhaseWorker) binExplore(area ExploreAreaOut) error {
	if area.Area.SizeX == area.Area.SizeY {
		leftArea, err := pw.game.client.PostExplore(area.Area.PosX, area.Area.PosY, area.Area.SizeX/2, area.Area.SizeY)
		if err != nil {
			return err
		}

		if leftArea.Amount == area.Amount {
			if leftArea.Area.SizeX == 1 && leftArea.Area.SizeY == 1 {
				pw.digAreas = append(pw.digAreas, leftArea)
				atomic.AddUint64(&pw.game.gameStats.SingleCellExploresWithTreasures, 1)
				return nil
			}

			return pw.binExplore(leftArea)
		}

		if leftArea.Area.SizeX == 1 && leftArea.Area.SizeY == 1 && leftArea.Amount > 0 {
			pw.digAreas = append(pw.digAreas, leftArea)
			atomic.AddUint64(&pw.game.gameStats.SingleCellExploresWithTreasures, 1)
		}

		rightArea, err := pw.game.client.PostExplore(area.Area.PosX+area.Area.SizeX/2, area.Area.PosY, area.Area.SizeX/2, area.Area.SizeY)
		if err != nil {
			return err
		}

		if rightArea.Area.SizeX == 1 && rightArea.Area.SizeY == 1 && rightArea.Amount > 0 {
			pw.digAreas = append(pw.digAreas, rightArea)
			atomic.AddUint64(&pw.game.gameStats.SingleCellExploresWithTreasures, 1)
			return nil
		}

		if leftArea.Amount > rightArea.Amount {
			if err := pw.binExplore(leftArea); err != nil {
				return err
			}

			if rightArea.Amount != 0 {
				return pw.binExplore(rightArea)
			}
		} else {
			if err := pw.binExplore(rightArea); err != nil {
				return err
			}

			if leftArea.Amount != 0 {
				return pw.binExplore(leftArea)
			}
		}

		return nil
	} else {
		topArea, err := pw.game.client.PostExplore(area.Area.PosX, area.Area.PosY+area.Area.SizeY/2, area.Area.SizeX, area.Area.SizeY/2)
		if err != nil {
			return err
		}

		if topArea.Amount == area.Amount {
			if topArea.Area.SizeX == 1 && topArea.Area.SizeY == 1 {
				pw.digAreas = append(pw.digAreas, topArea)
				atomic.AddUint64(&pw.game.gameStats.SingleCellExploresWithTreasures, 1)
				return nil
			}

			return pw.binExplore(topArea)
		}

		if topArea.Area.SizeX == 1 && topArea.Area.SizeY == 1 && topArea.Amount > 0 {
			pw.digAreas = append(pw.digAreas, topArea)
			atomic.AddUint64(&pw.game.gameStats.SingleCellExploresWithTreasures, 1)
		}

		bottomArea, err := pw.game.client.PostExplore(area.Area.PosX, area.Area.PosY, area.Area.SizeX, area.Area.SizeY/2)
		if err != nil {
			return err
		}

		if bottomArea.Area.SizeX == 1 && bottomArea.Area.SizeY == 1 && bottomArea.Amount > 0 {
			pw.digAreas = append(pw.digAreas, bottomArea)
			atomic.AddUint64(&pw.game.gameStats.SingleCellExploresWithTreasures, 1)
			return nil
		}

		if topArea.Amount > bottomArea.Amount {
			if err := pw.binExplore(topArea); err != nil {
				return err
			}

			if bottomArea.Amount != 0 {
				return pw.binExplore(bottomArea)
			}
		} else {
			if err := pw.binExplore(bottomArea); err != nil {
				return err
			}

			if topArea.Amount != 0 {
				return pw.binExplore(topArea)
			}
		}

		return nil
	}
}

func (pw *PhaseWorker) binExploreRow(area ExploreAreaOut) error {
	leftArea, err := pw.game.client.PostExplore(area.Area.PosX, area.Area.PosY, area.Area.SizeX/2, area.Area.SizeY)
	if err != nil {
		return err
	}

	// если ячейка не 1x1
	if leftArea.Area.SizeX != 1 {
		// если все сокровища в левой части, то исследуем только левую часть
		if leftArea.Amount == area.Amount {
			return pw.binExploreRow(leftArea)
		}

		// если все сокровища в правой части,  то исследуем только правую часть
		if leftArea.Amount == 0 {
			rightArea := leftArea
			rightArea.Area.PosX += leftArea.Area.SizeX
			rightArea.Amount = area.Amount

			return pw.binExploreRow(rightArea)
		}

		// если сокровища и в левой и в правой части, то исследуем обе части
		if err := pw.binExploreRow(leftArea); err != nil {
			return err
		}

		rightArea := leftArea
		rightArea.Area.PosX += leftArea.Area.SizeX
		rightArea.Amount = area.Amount - leftArea.Amount

		return pw.binExploreRow(rightArea)
	}

	// если ячейка 1x1

	// если все сокровища в левой ячейке, то сохраняем ячейку
	if leftArea.Amount == area.Amount {
		pw.digAreas = append(pw.digAreas, leftArea)
		atomic.AddUint64(&pw.game.gameStats.SingleCellExploresWithTreasures, 1)
		return nil
	}

	// если в левой ячейке есть сокровище, то сохраняем её
	if leftArea.Amount > 0 {
		pw.digAreas = append(pw.digAreas, leftArea)
		atomic.AddUint64(&pw.game.gameStats.SingleCellExploresWithTreasures, 1)
	}

	// т.к. в левой ячейке не все сокровища то правую ячейку тоже сохраняем
	rightArea := leftArea
	rightArea.Area.PosX += leftArea.Area.SizeX
	rightArea.Amount = area.Amount - leftArea.Amount

	pw.digAreas = append(pw.digAreas, rightArea)
	atomic.AddUint64(&pw.game.gameStats.SingleCellExploresWithTreasures, 1)

	return nil
}
