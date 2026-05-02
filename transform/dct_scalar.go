package transform

// IDCT4x4Scalar is the pure Go reference implementation.
func IDCT4x4Scalar(block []int16) {
	for i := 0; i < 4; i++ {
		row := block[i*4 : i*4+4]
		e0 := row[0] + row[2]
		e1 := row[0] - row[2]
		e2 := (row[1] >> 1) - row[3]
		e3 := row[1] + (row[3] >> 1)
		row[0] = e0 + e3
		row[1] = e1 + e2
		row[2] = e1 - e2
		row[3] = e0 - e3
	}
	for j := 0; j < 4; j++ {
		c0, c1, c2, c3 := block[j], block[4+j], block[8+j], block[12+j]
		e0 := c0 + c2
		e1 := c0 - c2
		e2 := (c1 >> 1) - c3
		e3 := c1 + (c3 >> 1)
		block[j] = (e0 + e3 + 32) >> 6
		block[4+j] = (e1 + e2 + 32) >> 6
		block[8+j] = (e1 - e2 + 32) >> 6
		block[12+j] = (e0 - e3 + 32) >> 6
	}
}

// DCT4x4Scalar is the pure Go reference implementation.
func DCT4x4Scalar(block []int16) {
	for i := 0; i < 4; i++ {
		row := block[i*4 : i*4+4]
		s0 := row[0] + row[3]
		s1 := row[1] + row[2]
		s2 := row[1] - row[2]
		s3 := row[0] - row[3]
		row[0] = s0 + s1
		row[1] = (s3 << 1) + s2
		row[2] = s0 - s1
		row[3] = s3 - (s2 << 1)
	}
	for j := 0; j < 4; j++ {
		c0, c1, c2, c3 := block[j], block[4+j], block[8+j], block[12+j]
		s0 := c0 + c3
		s1 := c1 + c2
		s2 := c1 - c2
		s3 := c0 - c3
		block[j] = s0 + s1
		block[4+j] = (s3 << 1) + s2
		block[8+j] = s0 - s1
		block[12+j] = s3 - (s2 << 1)
	}
}
