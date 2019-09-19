package math

// MinInt32 returns the smaller number of int32
func MinInt32(a, b int32) int32 {
	if a <= b {
		return a
	}

	return b
}

// MaxInt32 returns the bigger number of int32
func MaxInt32(a, b int32) int32 {
	if a >= b {
		return a
	}

	return b
}

// MinUint32 returns the smaller number of uint32
func MinUint32(a, b uint32) uint32 {
	if a <= b {
		return a
	}

	return b
}

// MaxUint32 returns the bigger number of uint32
func MaxUint32(a, b uint32) uint32 {
	if a >= b {
		return a
	}

	return b
}
