package mask

func Set(mask, flag byte) byte {
	return mask | flag
}

func Has(mask, flag byte) bool {
	return mask&flag == flag
}

func Clear(mask, flag byte) byte {
	return mask &^ flag
}
