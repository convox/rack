// Copyright 2012 Aryan Naraghi (aryan.naraghi@gmail.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package difflib

import (
	"reflect"
	"strings"
	"testing"
)

var lcsTests = []struct {
	seq1 string
	seq2 string
	lcs  int
}{
	{"", "", 0},
	{"abc", "abc", 3},
	{"mzjawxu", "xmjyauz", 4},
	{"human", "chimpanzee", 4},
	{"Hello, world!", "Hello, world!", 13},
	{"Hello, world!", "H     e    l  l o ,   w  o r l  d   !", 13},
}

func TestLongestCommonSubsequenceMatrix(t *testing.T) {
	for i, test := range lcsTests {
		seq1 := strings.Split(test.seq1, "")
		seq2 := strings.Split(test.seq2, "")
		matrix := longestCommonSubsequenceMatrix(seq1, seq2)
		lcs := matrix[len(matrix)-1][len(matrix[0])-1] // Grabs the lower, right value.
		if lcs != test.lcs {
			t.Errorf("%d. longestCommonSubsequence(%v, %v)[last][last] => %d, expected %d",
				i, seq1, seq2, lcs, test.lcs)
		}
	}
}

var numEqualStartAndEndElementsTests = []struct {
	seq1  string
	seq2  string
	start int
	end   int
}{
	{"", "", 0, 0},
	{"abc", "", 0, 0},
	{"", "abc", 0, 0},
	{"abc", "abc", 3, 0},
	{"abhelloc", "abbyec", 2, 1},
	{"abchello", "abcbye", 3, 0},
	{"helloabc", "byeabc", 0, 3},
}

func TestNumEqualStartAndEndElements(t *testing.T) {
	for i, test := range numEqualStartAndEndElementsTests {
		seq1 := strings.Split(test.seq1, "")
		seq2 := strings.Split(test.seq2, "")
		start, end := numEqualStartAndEndElements(seq1, seq2)
		if start != test.start || end != test.end {
			t.Errorf("%d. numEqualStartAndEndElements(%v, %v) => (%d, %d), expected (%d, %d)",
				i, seq1, seq2, start, end, test.start, test.end)
		}
	}
}

var diffTests = []struct {
	Seq1     string
	Seq2     string
	Diff     []DiffRecord
	HtmlDiff string
}{
	{
		"",
		"",
		[]DiffRecord{
			{"", Common},
		},
		`<tr><td class="line-num">1</td><td><pre></pre></td><td><pre></pre></td><td class="line-num">1</td></tr>
`,
	},

	{
		"same",
		"same",
		[]DiffRecord{
			{"same", Common},
		},
		`<tr><td class="line-num">1</td><td><pre>same</pre></td><td><pre>same</pre></td><td class="line-num">1</td></tr>
`,
	},

	{
		`one
two
three
`,
		`one
two
three
`,
		[]DiffRecord{
			{"one", Common},
			{"two", Common},
			{"three", Common},
			{"", Common},
		},
		`<tr><td class="line-num">1</td><td><pre>one</pre></td><td><pre>one</pre></td><td class="line-num">1</td></tr>
<tr><td class="line-num">2</td><td><pre>two</pre></td><td><pre>two</pre></td><td class="line-num">2</td></tr>
<tr><td class="line-num">3</td><td><pre>three</pre></td><td><pre>three</pre></td><td class="line-num">3</td></tr>
<tr><td class="line-num">4</td><td><pre></pre></td><td><pre></pre></td><td class="line-num">4</td></tr>
`,
	},

	{
		`one
two
three
`,
		`one
five
three
`,
		[]DiffRecord{
			{"one", Common},
			{"two", LeftOnly},
			{"five", RightOnly},
			{"three", Common},
			{"", Common},
		},
		`<tr><td class="line-num">1</td><td><pre>one</pre></td><td><pre>one</pre></td><td class="line-num">1</td></tr>
<tr><td class="line-num">2</td><td class="deleted"><pre>two</pre></td><td></td><td></td></tr>
<tr><td class="line-num"></td><td></td><td class="added"><pre>five</pre></td><td class="line-num">2</td></tr>
<tr><td class="line-num">3</td><td><pre>three</pre></td><td><pre>three</pre></td><td class="line-num">3</td></tr>
<tr><td class="line-num">4</td><td><pre></pre></td><td><pre></pre></td><td class="line-num">4</td></tr>
`,
	},

	{
		`Beethoven
Bach
Mozart
Chopin
`,
		`Beethoven
Bach
Brahms
Chopin
Liszt
Wagner
`,

		[]DiffRecord{
			{"Beethoven", Common},
			{"Bach", Common},
			{"Mozart", LeftOnly},
			{"Brahms", RightOnly},
			{"Chopin", Common},
			{"Liszt", RightOnly},
			{"Wagner", RightOnly},
			{"", Common},
		},
		`<tr><td class="line-num">1</td><td><pre>Beethoven</pre></td><td><pre>Beethoven</pre></td><td class="line-num">1</td></tr>
<tr><td class="line-num">2</td><td><pre>Bach</pre></td><td><pre>Bach</pre></td><td class="line-num">2</td></tr>
<tr><td class="line-num">3</td><td class="deleted"><pre>Mozart</pre></td><td></td><td></td></tr>
<tr><td class="line-num"></td><td></td><td class="added"><pre>Brahms</pre></td><td class="line-num">3</td></tr>
<tr><td class="line-num">4</td><td><pre>Chopin</pre></td><td><pre>Chopin</pre></td><td class="line-num">4</td></tr>
<tr><td class="line-num"></td><td></td><td class="added"><pre>Liszt</pre></td><td class="line-num">5</td></tr>
<tr><td class="line-num"></td><td></td><td class="added"><pre>Wagner</pre></td><td class="line-num">6</td></tr>
<tr><td class="line-num">5</td><td><pre></pre></td><td><pre></pre></td><td class="line-num">7</td></tr>
`,
	},

	{
		`adagio
vivace
staccato legato
presto
lento
`,
		`adagio adagio
staccato
staccato legato
staccato
legato
allegro
`,
		[]DiffRecord{
			{"adagio", LeftOnly},
			{"vivace", LeftOnly},
			{"adagio adagio", RightOnly},
			{"staccato", RightOnly},
			{"staccato legato", Common},
			{"presto", LeftOnly},
			{"lento", LeftOnly},
			{"staccato", RightOnly},
			{"legato", RightOnly},
			{"allegro", RightOnly},
			{"", Common},
		},
		`<tr><td class="line-num">1</td><td class="deleted"><pre>adagio</pre></td><td></td><td></td></tr>
<tr><td class="line-num">2</td><td class="deleted"><pre>vivace</pre></td><td></td><td></td></tr>
<tr><td class="line-num"></td><td></td><td class="added"><pre>adagio adagio</pre></td><td class="line-num">1</td></tr>
<tr><td class="line-num"></td><td></td><td class="added"><pre>staccato</pre></td><td class="line-num">2</td></tr>
<tr><td class="line-num">3</td><td><pre>staccato legato</pre></td><td><pre>staccato legato</pre></td><td class="line-num">3</td></tr>
<tr><td class="line-num">4</td><td class="deleted"><pre>presto</pre></td><td></td><td></td></tr>
<tr><td class="line-num">5</td><td class="deleted"><pre>lento</pre></td><td></td><td></td></tr>
<tr><td class="line-num"></td><td></td><td class="added"><pre>staccato</pre></td><td class="line-num">4</td></tr>
<tr><td class="line-num"></td><td></td><td class="added"><pre>legato</pre></td><td class="line-num">5</td></tr>
<tr><td class="line-num"></td><td></td><td class="added"><pre>allegro</pre></td><td class="line-num">6</td></tr>
<tr><td class="line-num">6</td><td><pre></pre></td><td><pre></pre></td><td class="line-num">7</td></tr>
`,
	},
}

func TestDiff(t *testing.T) {
	for i, test := range diffTests {
		seq1 := strings.Split(test.Seq1, "\n")
		seq2 := strings.Split(test.Seq2, "\n")

		diff := Diff(seq1, seq2)
		if !reflect.DeepEqual(diff, test.Diff) {
			t.Errorf("%d. Diff(%v, %v) => %v, expected %v",
				i, seq1, seq2, diff, test.Diff)
		}

		htmlDiff := HTMLDiff(seq1, seq2)
		if htmlDiff != test.HtmlDiff {
			t.Errorf("%d. HtmlDiff(%v, %v) => %v, expected %v",
				i, seq1, seq2, htmlDiff, test.HtmlDiff)
		}

	}
}
