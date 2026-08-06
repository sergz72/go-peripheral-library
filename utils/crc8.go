package utils

type Crc8 struct {
	polynom byte
}

func (c *Crc8) Compute(data []byte) byte {
	// initialization value
	var crc byte = 0xff

	// iterate over all bytes
	for _, d := range data {
		crc ^= d
		for range 8 {
			xor := crc & 0x80
			crc = crc << 1
			if xor != 0 {
				crc ^= c.polynom
			}
		}
	}

	return crc
}

func NewCrc31() *Crc8 {
	return &Crc8{0x31}
}
