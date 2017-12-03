package dynamic_test

import (
	"github.com/twmb/algoimpl/go/various/dynamic"
	"testing"
)

func TestCutRod(t *testing.T) {
	prices := make([]int, 11)
	prices[1] = 1
	prices[2] = 5
	prices[3] = 8
	prices[4] = 9
	prices[5] = 10
	prices[6] = 17
	prices[7] = 17
	prices[8] = 20
	prices[9] = 24
	prices[10] = 30

	idealProfits := make([]int, 11)
	idealProfits[1] = 1
	idealProfits[2] = 5
	idealProfits[3] = 8
	idealProfits[4] = 10
	idealProfits[5] = 13
	idealProfits[6] = 17
	idealProfits[7] = 18
	idealProfits[8] = 22
	idealProfits[9] = 25
	idealProfits[10] = 30

	dynamic.InitPrices(prices)
	for i := len(prices) - 1; i >= 0; i-- {
		got := dynamic.CutRod(i)
		if got.Profit != idealProfits[i] {
			t.Errorf("Expected profit %v, got profit %v with positions %v", idealProfits[i], got.Profit, got.CutPositions)
		}
	}
}
