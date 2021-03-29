package client

import (
	"fmt"
	"sync"
	"time"
)

type GameStats struct {
	FreeLicenses                    uint64
	PaidLicenses                    uint64
	ExpensiveLicenses               uint64
	SingleCellExplores              uint64
	SingleCellExploresWithTreasures uint64
	TreasuresFound                  uint64
	TreasuresExchanged              uint64
	TreasuresExchangeFailed         uint64
	LineCost                        int
	exploreCost                     float64
	exploreCostLock                 *sync.Mutex
	digCost                         float64
	digCostLock                     *sync.Mutex
	cashCost                        float64
	cashCostLock                    *sync.Mutex
	licenseWait                     int64
	licenseWaitLock                 *sync.Mutex
}

func (gs *GameStats) printStats() {
	fmt.Printf("Free licenses: %d\n", gs.FreeLicenses)
	fmt.Printf("Paid licenses: %d\n", gs.PaidLicenses)
	fmt.Printf("Expensive licenses: %d\n", gs.ExpensiveLicenses)
	fmt.Printf("Wait licenses ms: %d\n", gs.licenseWait)

	fmt.Printf("Single cell explores: %d\n", gs.SingleCellExplores)
	fmt.Printf("Single cell explores with treasures: %d\n", gs.SingleCellExploresWithTreasures)

	fmt.Printf("Treasures found: %d\n", gs.TreasuresFound)
	fmt.Printf("Treasures exchanged: %d\n", gs.TreasuresExchanged)
	fmt.Printf("Treasures not exchanged: %d\n", gs.TreasuresFound-gs.TreasuresExchanged-gs.TreasuresExchangeFailed)
	fmt.Printf("Treasures exchange failed: %d\n", gs.TreasuresExchangeFailed)

	fmt.Printf("Explore cost: %f\n", gs.exploreCost)
	fmt.Printf("Explore cost per sec: %f\n", gs.exploreCost/598)

	fmt.Printf("Dig cost: %f\n", gs.digCost)
	fmt.Printf("Dig cost per sec: %f\n", gs.digCost/598)

	fmt.Printf("Cash cost: %f\n", gs.cashCost)
	fmt.Printf("Cash cost per sec: %f\n", gs.cashCost/598)

	fmt.Printf("Total cost: %f\n", gs.exploreCost+gs.digCost+gs.digCost)
	fmt.Printf("Cost per sec: %f\n", (gs.exploreCost+gs.digCost+gs.digCost)/598)
}

func (gs *GameStats) writeDigCost(depth int) {
	gs.digCostLock.Lock()
	defer gs.digCostLock.Unlock()

	gs.digCost += 1.8 + float64(depth-1)*0.2
}

func (gs *GameStats) writeCashCost() {
	gs.cashCostLock.Lock()
	defer gs.cashCostLock.Unlock()

	gs.cashCost += 20.0
}

func (gs *GameStats) writeLicenseWait(duration time.Duration) {
	gs.licenseWaitLock.Lock()
	defer gs.licenseWaitLock.Unlock()

	gs.licenseWait += duration.Milliseconds()
}

func (gs *GameStats) writeExploreCost(areaSize int) {
	gs.exploreCostLock.Lock()
	defer gs.exploreCostLock.Unlock()

	gs.exploreCost += gs.getExploreCost(areaSize)
}

func (gs *GameStats) getExploreCost(areaSize int) float64 {
	switch {
	case areaSize < 4:
		return 1
	case areaSize < 8:
		return 2
	case areaSize < 16:
		return 3
	case areaSize < 32:
		return 4
	case areaSize < 64:
		return 5
	case areaSize < 128:
		return 6
	default:
		return 7
	}
}
