package scanner

import "math"

// ShannonEntropy calculates the Shannon entropy of a string in bits per character.
func ShannonEntropy(data string) float64 {
	if data == "" {
		return 0
	}
	charCounts := make(map[rune]int)
	for _, char := range data {
		charCounts[char]++
	}
	var entropy float64
	for _, count := range charCounts {
		freq := float64(count) / float64(len(data))
		entropy -= freq * math.Log2(freq)
	}
	return entropy
}

