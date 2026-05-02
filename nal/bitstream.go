package nal

// Bitstream reader for H.264 exp-Golomb and fixed-length codes.
// All H.264 syntax elements are read through this interface.

// Reader reads bits from a byte slice with H.264 emulation prevention.
type Reader struct {
	data []byte
	pos  int // byte position
	bit  int // bit position within current byte (7 = MSB, 0 = LSB)
}

// NewReader creates a bitstream reader over raw NAL unit payload (after start code + header).
func NewReader(data []byte) *Reader {
	return &Reader{data: data, pos: 0, bit: 7}
}

// ReadBit reads a single bit.
func (r *Reader) ReadBit() uint32 {
	if r.pos >= len(r.data) {
		return 0
	}
	v := uint32((r.data[r.pos] >> uint(r.bit)) & 1)
	r.bit--
	if r.bit < 0 {
		r.bit = 7
		r.pos++
		// Emulation prevention: skip 0x03 in 0x00 0x00 0x03
		if r.pos >= 2 && r.pos < len(r.data) &&
			r.data[r.pos-2] == 0 && r.data[r.pos-1] == 0 && r.data[r.pos] == 3 {
			r.pos++
		}
	}
	return v
}

// ReadBits reads n bits (up to 32) as a uint32.
func (r *Reader) ReadBits(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = (v << 1) | r.ReadBit()
	}
	return v
}

// ReadUE reads an unsigned exp-Golomb coded value.
// Format: leading zeros, 1, suffix bits.
// 0 → 0, 010 → 1, 011 → 2, 00100 → 3, etc.
func (r *Reader) ReadUE() uint32 {
	zeros := 0
	for r.ReadBit() == 0 {
		zeros++
		if zeros > 31 {
			return 0 // overflow protection
		}
	}
	if zeros == 0 {
		return 0
	}
	return (1 << uint(zeros)) - 1 + r.ReadBits(zeros)
}

// ReadSE reads a signed exp-Golomb coded value.
// Mapping: 0→0, 1→1, 2→-1, 3→2, 4→-2, etc.
func (r *Reader) ReadSE() int32 {
	v := r.ReadUE()
	if v%2 == 0 {
		return -int32(v / 2)
	}
	return int32((v + 1) / 2)
}

// ReadBool reads a single bit as a boolean (u(1)).
func (r *Reader) ReadBool() bool {
	return r.ReadBit() != 0
}

// ReadU8 reads an 8-bit unsigned integer.
func (r *Reader) ReadU8() uint8 {
	return uint8(r.ReadBits(8))
}

// EOF returns true if the reader has consumed all data.
func (r *Reader) EOF() bool {
	return r.pos >= len(r.data)
}

// ByteAligned returns true if the current position is byte-aligned.
func (r *Reader) ByteAligned() bool {
	return r.bit == 7
}

// ByteAlign skips bits until the next byte boundary.
func (r *Reader) ByteAlign() {
	if r.bit != 7 {
		r.bit = 7
		r.pos++
	}
}
