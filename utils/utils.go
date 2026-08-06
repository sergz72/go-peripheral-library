package utils

import "bytes"

func ZeroTerminatedToString(byteArray []byte) string {
	nullIndex := bytes.IndexByte(byteArray, 0) // Find the first occurrence of the null byte (\x00)
	if nullIndex == -1 {
		return string(byteArray) // No null terminator found, convert entire slice
	}
	return string(byteArray[:nullIndex]) // Slice up to the null terminator
}
