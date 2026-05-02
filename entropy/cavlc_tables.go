package entropy

// CAVLC coeff_token VLC tables (ITU-T H.264 Tables 9-5a through 9-5d).
// Each entry: (totalCoeff, trailingOnes, codeLength)
// Indexed by table number (0-3) based on nC range.

// CoeffToken represents a decoded coeff_token.
type CoeffToken struct {
	TotalCoeff   int
	TrailingOnes int
}

// nC ranges: 0=0..1, 1=2..3, 2=4..7, 3=8+
// For table 3 (nC >= 8): fixed-length 6-bit code.

// Table 0 (nC = 0..1): most common in practice
// Format: binary prefix tree. We use a flat lookup with max 16 bits.
// Entries sorted by code value.
var coeffTokenTable0 = []struct {
	bits   uint32
	length int
	tc, to int // totalCoeff, trailingOnes
}{
	{0b1, 1, 0, 0},
	{0b000101, 6, 1, 0},
	{0b01, 2, 1, 1},
	{0b00000111, 8, 2, 0},
	{0b000100, 6, 2, 1},
	{0b001, 3, 2, 2},
	{0b000000111, 9, 3, 0},
	{0b00000110, 8, 3, 1},
	{0b0000101, 7, 3, 2},
	{0b00011, 5, 3, 3},
	{0b0000000111, 10, 4, 0},
	{0b000000110, 9, 4, 1},
	{0b00000101, 8, 4, 2},
	{0b000011, 6, 4, 3},
}

// For the full spec tables, a proper VLC decoder tree would be used.
// This simplified version handles the most common cases.
