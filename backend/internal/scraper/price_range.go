package scraper

import (
	"fmt"
	"math"
	"sort"
)

func FormatPriceRange(pricesRUB, pricesUSD []float64, exchangeRate float64) string {
	if len(pricesRUB) == 0 && len(pricesUSD) == 0 {
		return ""
	}

	var allPricesRUB, allPricesUSD []float64

	if len(pricesRUB) > 0 {
		for _, p := range pricesRUB {
			if p > 0 {
				allPricesRUB = append(allPricesRUB, p)
			}
		}
	}

	if len(pricesUSD) > 0 {
		for _, p := range pricesUSD {
			if p > 0 {
				allPricesUSD = append(allPricesUSD, p)
			}
		}
	}

	if len(allPricesRUB) == 0 && len(allPricesUSD) == 0 {
		return ""
	}

	if len(allPricesRUB) == 0 {
		allPricesRUB = convertUSDToRUB(allPricesUSD, exchangeRate)
	}
	if len(allPricesUSD) == 0 {
		allPricesUSD = convertRUBToUSD(allPricesRUB, exchangeRate)
	}

	if len(allPricesRUB) > 0 {
		sort.Float64s(allPricesRUB)
	}
	if len(allPricesUSD) > 0 {
		sort.Float64s(allPricesUSD)
	}

	minRUB := allPricesRUB[0]
	maxRUB := allPricesRUB[len(allPricesRUB)-1]
	minUSD := allPricesUSD[0]
	maxUSD := allPricesUSD[len(allPricesUSD)-1]

	usdRange := formatPrice(fmt.Sprintf("%.0f - %.0f USD", minUSD, maxUSD))
	rubRange := formatPrice(fmt.Sprintf("%.0f - %.0f RUB", minRUB, maxRUB))

	return fmt.Sprintf("%s / %s", usdRange, rubRange)
}

func convertUSDToRUB(pricesUSD []float64, exchangeRate float64) []float64 {
	result := make([]float64, len(pricesUSD))
	for i, p := range pricesUSD {
		result[i] = p * exchangeRate
	}
	return result
}

func convertRUBToUSD(pricesRUB []float64, exchangeRate float64) []float64 {
	result := make([]float64, len(pricesRUB))
	for i, p := range pricesRUB {
		result[i] = p / exchangeRate
	}
	return result
}

func formatPrice(s string) string {
	var result []byte
	j := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] >= '0' && s[i] <= '9' {
			if j > 0 && j%3 == 0 {
				result = append([]byte{' '}, result...)
			}
			result = append([]byte{s[i]}, result...)
			j++
		} else {
			result = append([]byte{s[i]}, result...)
			j = 0
		}
	}
	return string(result)
}

func roundToInt(val float64) int {
	return int(math.Round(val))
}
