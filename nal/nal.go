package nal

import "fmt"

// NAL unit types (ITU-T H.264 Table 7-1)
const (
	TypeSliceNonIDR  = 1  // Coded slice of a non-IDR picture
	TypeSlicePartA   = 2  // Coded slice data partition A
	TypeSlicePartB   = 3  // Coded slice data partition B
	TypeSlicePartC   = 4  // Coded slice data partition C
	TypeSliceIDR     = 5  // Coded slice of an IDR picture
	TypeSEI          = 6  // Supplemental enhancement information
	TypeSPS          = 7  // Sequence parameter set
	TypePPS          = 8  // Picture parameter set
	TypeAUD          = 9  // Access unit delimiter
	TypeEndSeq       = 10 // End of sequence
	TypeEndStream    = 11 // End of stream
	TypeFiller       = 12 // Filler data
)

// Unit is a parsed NAL unit.
type Unit struct {
	RefIDC   uint8  // nal_ref_idc (2 bits): reference priority
	Type     uint8  // nal_unit_type (5 bits)
	Payload  []byte // RBSP payload (after header, before next start code)
}

// IsSlice returns true if this NAL contains slice data.
func (u *Unit) IsSlice() bool {
	return u.Type >= 1 && u.Type <= 5
}

// TypeName returns a human-readable name for the NAL type.
func (u *Unit) TypeName() string {
	switch u.Type {
	case TypeSliceNonIDR: return "Slice"
	case TypeSliceIDR:    return "IDR"
	case TypeSPS:         return "SPS"
	case TypePPS:         return "PPS"
	case TypeSEI:         return "SEI"
	case TypeAUD:         return "AUD"
	default:              return fmt.Sprintf("NAL(%d)", u.Type)
	}
}

// SplitNALUnits splits an Annex B bitstream into NAL units.
// Annex B format: [0x00 0x00 0x01 | 0x00 0x00 0x00 0x01] <NAL bytes>
func SplitNALUnits(data []byte) []Unit {
	var units []Unit
	n := len(data)
	i := 0

	// Find first start code
	for i < n-3 {
		if data[i] == 0 && data[i+1] == 0 {
			if data[i+2] == 1 {
				i += 3
				break
			}
			if i < n-3 && data[i+2] == 0 && data[i+3] == 1 {
				i += 4
				break
			}
		}
		i++
	}

	for i < n {
		// Parse NAL header byte
		header := data[i]
		refIDC := (header >> 5) & 0x3
		nalType := header & 0x1F
		i++

		// Find next start code or end of data
		start := i
		for i < n-3 {
			if data[i] == 0 && data[i+1] == 0 {
				if data[i+2] == 1 {
					break
				}
				if i < n-3 && data[i+2] == 0 && data[i+3] == 1 {
					break
				}
			}
			i++
		}
		if i >= n-3 {
			i = n
		}

		// Trim trailing zeros before next start code
		end := i
		for end > start && data[end-1] == 0 {
			end--
		}

		units = append(units, Unit{
			RefIDC:  refIDC,
			Type:    nalType,
			Payload: data[start:end],
		})

		// Skip start code
		if i < n-3 {
			if data[i+2] == 1 {
				i += 3
			} else {
				i += 4
			}
		}
	}

	return units
}
