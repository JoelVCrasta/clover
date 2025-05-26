package client

type Bitfield []byte

// Has checks if the bit at the given index in the bitfield is set to 1.
func (b Bitfield) Has(index int) bool {
	byteIndex := index / 8
	if byteIndex < 0 || byteIndex >= len(b) {
		return false
	}

	bitIndex := index % 8
	return b[byteIndex]>>(7-bitIndex)&1 == 1
}

// Set sets the bit at the given index in the bitfield to 1.
func (b Bitfield) Set(index int) {
	byteIndex := index / 8
	if byteIndex < 0 || byteIndex >= len(b) {
		return
	}

	bitIndex := index % 8
	b[byteIndex] |= (1 << (7 - bitIndex))
}
