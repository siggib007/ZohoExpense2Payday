package main

import "strings"

func ValidateKT(strKennitala string) bool {
	strKennitala = strings.NewReplacer("-", "", " ", "", ".", "", ",", "").Replace(strKennitala)

	if len(strKennitala) != 10 {
		return false
	}
	for _, cChar := range strKennitala {
		if cChar < '0' || cChar > '9' {
			return false
		}
	}

	lstCheck := []int{3, 2, 7, 6, 5, 4, 3, 2}
	iSum := 0
	for iIndex, iWeight := range lstCheck {
		iDigit := int(strKennitala[iIndex] - '0')
		iSum += iDigit * iWeight
	}

	iCheck := iSum % 11
	iExpected := 11 - iCheck
	return iExpected == int(strKennitala[8]-'0')
}
