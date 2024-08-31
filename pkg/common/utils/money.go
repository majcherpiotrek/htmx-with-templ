package utils

import "github.com/shopspring/decimal"

func NullDecimalFromFloat64(f *float64) decimal.NullDecimal {
	if f != nil {
		d := decimal.NewFromFloat(*f)
		return decimal.NewNullDecimal(d)
	}

	return decimal.NullDecimal{
		Decimal: decimal.Zero,
		Valid:   false,
	}
}
