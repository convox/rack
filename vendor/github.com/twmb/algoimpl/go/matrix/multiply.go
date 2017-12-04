package matrix

import (
	"errors"
)

// I do not check if any row length differs from others: don't do that.
// You know who does that? Crazy people.

func testBounds(A, B [][]int) error {
	if len(A) == 0 || len(B) == 0 {
		return errors.New("Cannot multiply empty matrices")
	}
	if len(A[0]) == 0 || len(B[0]) == 0 {
		return errors.New("Cannot multiply empty matrices")
	}
	if len(A[0]) != len(B) {
		return errors.New("Dimension mismatch")
	}
	return nil
}

func BasicMultiply(A, B [][]int) ([][]int, error) {
	err := testBounds(A, B)
	if err != nil {
		return nil, err
	}
	C := make([][]int, len(A))
	for r := range A {
		C[r] = make([]int, len(B[0]))
		for c := range B[0] {
			for k := range A[r] {
				C[r][c] += A[r][k] * B[k][c]
			}
		}
	}
	return C, nil
}

type multBounds struct {
	rs, re, cs, ce int // {row,column}{start,end}
}

// pad to power of two and deep copy the elements
func pad(M [][]int, power int) [][]int {
	newM := make([][]int, power)
	for i := 0; i < len(M); i++ {
		newM[i] = make([]int, power)
		copy(newM[i], M[i])
	}
	for i := len(M); i < power; i++ {
		newM[i] = make([]int, power)
	}
	return newM
}

func add(s1, s2, t [][]int, boundsT multBounds) {
	for j := 0; j < len(s1); j++ {
		for i := 0; i < len(s1[0]); i++ {
			t[boundsT.rs+j][boundsT.cs+i] += s1[j][i] + s2[j][i]
		}
	}
}

func recursiveMultiplyImpl(A, B, C [][]int, boundsA, boundsB multBounds) [][]int {
	if boundsA.re-boundsA.rs <= 1 {
		return [][]int{[]int{A[boundsA.rs][boundsA.cs] * B[boundsB.rs][boundsB.cs]}}
	} else {
		// currently these will be equal, if I modify the recursive function to not pad to equal power lengths of two, this will be more relevant
		tla := multBounds{boundsA.rs, (boundsA.rs + boundsA.re) / 2, boundsA.cs, (boundsA.cs + boundsA.ce) / 2} // top left a
		tra := multBounds{boundsA.rs, (boundsA.rs + boundsA.re) / 2, (boundsA.cs + boundsA.ce) / 2, boundsA.ce} // top right a
		bla := multBounds{(boundsA.rs + boundsA.re) / 2, boundsA.re, boundsA.cs, (boundsA.cs + boundsA.ce) / 2} // bottom left a
		bra := multBounds{(boundsA.rs + boundsA.re) / 2, boundsA.re, (boundsA.cs + boundsA.ce) / 2, boundsA.ce} // bottom right a

		tlb := multBounds{boundsB.rs, (boundsB.rs + boundsB.re) / 2, boundsB.cs, (boundsB.cs + boundsB.ce) / 2} // top left b
		trb := multBounds{boundsB.rs, (boundsB.rs + boundsB.re) / 2, (boundsB.cs + boundsB.ce) / 2, boundsB.ce} // top right b
		blb := multBounds{(boundsB.rs + boundsB.re) / 2, boundsB.re, boundsB.cs, (boundsB.cs + boundsB.ce) / 2} // bottom left b
		brb := multBounds{(boundsB.rs + boundsB.re) / 2, boundsB.re, (boundsB.cs + boundsB.ce) / 2, boundsB.ce} // bottom right b

		newC := make([][]int, boundsA.re-boundsA.rs)
		for j := range newC {
			newC[j] = make([]int, boundsA.ce-boundsA.cs)
			for i := range newC[j] {
				newC[j][i] = C[boundsA.rs+j][boundsA.cs+i]
			}
		}

		tlc := multBounds{0, len(newC) / 2, 0, len(newC[0]) / 2}
		trc := multBounds{len(newC) / 2, len(newC), 0, len(newC[0]) / 2}
		blc := multBounds{0, len(newC) / 2, len(newC[0]) / 2, len(newC[0])}
		brc := multBounds{len(newC) / 2, len(newC), len(newC[0]) / 2, len(newC[0])}

		r1 := recursiveMultiplyImpl(A, B, C, tla, tlb)
		r2 := recursiveMultiplyImpl(A, B, C, tra, blb)
		add(r1, r2, newC, tlc)

		r3 := recursiveMultiplyImpl(A, B, C, tla, trb)
		r4 := recursiveMultiplyImpl(A, B, C, tra, brb)
		add(r3, r4, newC, trc)

		r5 := recursiveMultiplyImpl(A, B, C, bla, tlb)
		r6 := recursiveMultiplyImpl(A, B, C, bra, blb)
		add(r5, r6, newC, blc)

		r7 := recursiveMultiplyImpl(A, B, C, bla, trb)
		r8 := recursiveMultiplyImpl(A, B, C, bra, brb)
		add(r7, r8, newC, brc)
		return newC
	}
}

func RecursiveMultiply(A, B [][]int) ([][]int, error) {
	// Note: I *really* hate recursive matrix multiplication. So, you can do it on any matrix, but I have not yet
	// made this efficient (even if it's O3, I still have many constant O2 additions that I don't need). I have
	// a mix of ideas in getting this to work. Ideally, I would only use one C matrix and do operations based off
	// of relative indices. But I don't have that yet. I both duplicate smaller boxes many times and pass around C.
	// So it's not smart. But it works!
	// So, the TODO list is:
	//  1) Take out the duplications on C boxes
	//  2) Take out the padding for the multipication and work magically with the numbers
	//  3) Implement Strassen's algorithm
	//
	// Yet again, I'm probably not going to do any of these unless I get *really* bored some day because I find this algorithm
	// so boring and somewhat needlessly complex to implement (with slices, though I could switch to arrays and maybe it'd be
	// easier that way. Hmm...).
	err := testBounds(A, B)
	if err != nil {
		return nil, err
	}
	C := make([][]int, len(A))
	for i := range C {
		C[i] = make([]int, len(B[0]))
	}
	longest := len(A)
	if len(A[0]) > longest {
		longest = len(A[0])
	}
	if len(B) > longest {
		longest = len(B)
	}
	power := 1
	for power < longest {
		power <<= 1
	}
	newA := pad(A, power)
	newB := pad(B, power)
	newC := pad(C, power)
	boundsA := multBounds{
		0, len(newA[0]), 0, len(newA),
	}
	boundsB := multBounds{
		0, len(newB[0]), 0, len(newB),
	}
	newC = recursiveMultiplyImpl(newA, newB, newC, boundsA, boundsB)
	C = newC[0:len(A)]
	for i := range C {
		C[i] = newC[i][0:len(B[0])]
	}
	return C, nil
}
