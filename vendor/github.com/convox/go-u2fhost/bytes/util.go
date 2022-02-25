package bytes

import "errors"

// Concats the provided slices into a single slice.
func Concat(slices ...[]byte) []byte {
	var length int
	for _, s := range slices {
		length += len(s)
	}
	destination := make([]byte, length)
	ConcatInto(destination, slices...)
	return destination
}

// Concats the provided slices into the destination slice.
// Returns an error if the destination is smaller than the combined
// length of the provided slices.
func ConcatInto(destination []byte, slices ...[]byte) ([]byte, error) {
	if destination == nil {
		return nil, errors.New("Destination slice cannot be nil")
	}
	var offset int
	for _, slice := range slices {
		if len(slice)+offset > len(destination) {
			return nil, errors.New("Destination slice is to small to contain provided slices")
		}
		copy(destination[offset:], slice)
		offset += len(slice)
	}
	return destination, nil
}
