package pred

// PredIntra8x8 generates the predicted 8×8 block from neighboring pixels.
// Uses the same 9 modes as Intra4x4 but applied to 8×8 blocks.
// top: 8+1 pixels above (+1 for filtering), left: 8 pixels, topLeft: corner.
func PredIntra8x8(pred []uint8, mode int, top, left []uint8, topLeft uint8) {
	switch mode {
	case Intra4x4Vertical:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				pred[y*8+x] = top[x]
			}
		}
	case Intra4x4Horizontal:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				pred[y*8+x] = left[y]
			}
		}
	case Intra4x4DC:
		sum := uint16(0)
		for i := 0; i < 8; i++ {
			sum += uint16(top[i]) + uint16(left[i])
		}
		dc := uint8((sum + 8) >> 4)
		for i := range pred[:64] {
			pred[i] = dc
		}
	default:
		// Other modes: fall back to DC
		sum := uint16(0)
		for i := 0; i < 8; i++ {
			sum += uint16(top[i]) + uint16(left[i])
		}
		dc := uint8((sum + 8) >> 4)
		for i := range pred[:64] {
			pred[i] = dc
		}
	}
}
