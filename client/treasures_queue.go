package client

type TreasuresQueue struct {
	Treasures []*PriorityTreasure
}

func NewTreasuresQueue(size int) *TreasuresQueue {
	return &TreasuresQueue{
		Treasures: make([]*PriorityTreasure, 0, size),
	}
}

func (tq *TreasuresQueue) Len() int {
	return len(tq.Treasures)
}

func (tq *TreasuresQueue) Less(i, j int) bool {
	return tq.Treasures[i].Priority > tq.Treasures[j].Priority
}

func (tq *TreasuresQueue) Swap(i, j int) {
	tq.Treasures[i], tq.Treasures[j] = tq.Treasures[j], tq.Treasures[i]
	tq.Treasures[i].Index = i
	tq.Treasures[j].Index = j
}

func (tq *TreasuresQueue) Push(x interface{}) {
	n := len(tq.Treasures)
	item := x.(*PriorityTreasure)
	item.Index = n
	tq.Treasures = append(tq.Treasures, item)
}

func (tq *TreasuresQueue) Pop() interface{} {
	old := tq.Treasures
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	tq.Treasures = old[0 : n-1]
	return item
}
