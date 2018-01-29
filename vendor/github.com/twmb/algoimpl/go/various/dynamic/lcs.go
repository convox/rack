package dynamic

// Returns the longest common subsequence of two strings in O(mn) time
// and space complexity, where m and n are the lengths of the strings.
func LongestCommonSubsequence(first, second string) string {

	firstRunes := []rune(first)
	secondRunes := []rune(second)

	// Create lookup table - sides initialized to zero already
	table := make([][]int, 1+len(firstRunes))
	for i := range table {
		table[i] = make([]int, 1+len(secondRunes))
	}

	for i := 0; i < len(firstRunes); i++ {
		for j := 0; j < len(secondRunes); j++ {
			if firstRunes[i] == secondRunes[j] {
				table[i+1][j+1] = 1 + table[i][j]
			} else { // +1 to table indices because the zero row/column corresponds to no letters
				table[i+1][j+1] = max(table[i+1][j], table[i][j+1]) // favor longer first string
			}
		}
	}

	lcsRunes := make([]rune, 0)

	for i := len(firstRunes); i > 0; {
		for j := len(secondRunes); i > 0; {
			if table[i][j] == table[i-1][j] { // favored longer first string
				i-- // letters at {i,j} not equal, but equal at {i-1, j}
			} else if table[i][j] == table[i][j-1] {
				j-- // letters at {i,j} not equal, {i-1, j} not equal, equal at {i, j-1}
			} else {
				i-- // letters at {i,j} equal
				j--
				lcsRunes = append(lcsRunes, firstRunes[i])
			}
		}
	}

	// lcsRunes is backwards, reverse
	runelen := len(lcsRunes)
	for i := 0; i < runelen/2; i++ {
		lcsRunes[i], lcsRunes[runelen-i-1] = lcsRunes[runelen-i-1], lcsRunes[i]
	}
	return string(lcsRunes)
}

// returns the greater of the two ints, favoring the first if they are equal
func max(first, second int) int {
	if first >= second {
		return first
	}
	return second
}
