package dynamic

type MoneyRod struct {
	CutPositions []int
	Profit       int
}

var profits []int
var firstCuts []int
var profitsCalculatedIndex int

func InitPrices(prices []int) {
	profits = prices
	firstCuts = make([]int, len(profits))
	for i := range firstCuts {
		firstCuts[i] = i
	}
	profitsCalculatedIndex = 0
}

// Assumes that the 0 index of prices corresponds to a rod of 0 length (and will have a price of 0)
func CutRod(rodlength int) MoneyRod {
	if rodlength > profitsCalculatedIndex {
		if rodlength > len(profits) {
			return MoneyRod{}
		}
		for j := 1; j <= rodlength; j++ {
			for i := 1; i <= j/2; i++ {
				if profits[j] < profits[j-i]+profits[i] {
					firstCuts[j] = j - i
					profits[j] = profits[j-i] + profits[i]
				}
			}
		}
		profitsCalculatedIndex = rodlength
	}

	maxProfit := profits[rodlength]
	cutPositions := make([]int, 0)
	for rodlength > 0 {
		cutPositions = append(cutPositions, rodlength)
		rodlength -= firstCuts[rodlength]
	}
	return MoneyRod{CutPositions: cutPositions, Profit: maxProfit}
}
