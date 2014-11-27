// Copyright 2014 Rob Pike. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value // import "robpike.io/ivy/value"

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

type BigRat struct {
	*big.Rat
}

func setBigRatString(s string) (br BigRat, err error) {
	base := conf.InputBase()
	r := big.NewRat(0, 1)
	var ok bool
	slash := strings.IndexByte(s, '/')
	if slash < 0 {
		r, ok = r.SetString(s)
	} else {
		switch base {
		case 0, 10: // Fine as is.
			r, ok = r.SetString(s)
		default:
			// big.Rat doesn't handle arbitrary bases, but big.Int does,
			// so do the numerator and denominator separately.
			var num, denom BigInt
			num, err = setBigIntString(s[:slash])
			if err == nil {
				denom, err = setBigIntString(s[slash+1:])
			}
			if err != nil {
				return
			}
			return BigRat{r.SetFrac(num.Int, denom.Int)}, nil
		}
	}
	if !ok {
		return BigRat{}, errors.New("rational number syntax")
	}
	return BigRat{r}, nil
}

func (r BigRat) String() string {
	format := conf.Format()
	if format != "" {
		verb, prec, ok := conf.FloatFormat()
		if ok {
			return r.floatString(verb, prec)
		}
		return fmt.Sprintf(conf.RatFormat(), r.Num(), r.Denom())
	}
	num := BigInt{r.Num()}
	den := BigInt{r.Denom()}
	return fmt.Sprintf("%s/%s", num, den)
}

func (r BigRat) floatString(verb byte, prec int) string {
	switch verb {
	case 'f', 'F':
		return r.Rat.FloatString(prec)
	case 'e', 'E':
		// The exponent will alway be >= 0.
		sign := ""
		var x, t big.Rat
		x.Set(r.Rat)
		if x.Sign() < 0 {
			sign = "-"
			x.Neg(&x)
		}
		t.Set(&x)
		exp := ratExponent(&x)
		ratScale(&t, exp)
		str := t.FloatString(prec + 1) // +1 because first digit might be zero.
		// Drop the decimal.
		if str[0] == '0' {
			str = str[2:]
			exp--
		} else if len(str) > 1 && str[1] == '.' {
			str = str[0:1] + str[2:]
		}
		return eFormat(verb, prec, sign, str, exp)
	default:
		Errorf("can't handle verb %c for rational", verb)
	}
	return ""
}

var bigRatTen = big.NewRat(10, 1)
var bigRatBillion = big.NewRat(1e9, 1)

// ratExponent returns the power of ten that x would display in scientific notation.
func ratExponent(x *big.Rat) int {
	invert := false
	if x.Num().Cmp(x.Denom()) < 0 {
		invert = true
		x.Inv(x)
	}
	e := 0
	for x.Cmp(bigRatBillion) >= 0 {
		e += 9
		x.Quo(x, bigRatBillion)
	}
	for x.Cmp(bigRatTen) >= 0 {
		e++
		x.Quo(x, bigRatTen)
	}
	if invert {
		return -e
	}
	return e
}

// ratScale multiplies x by 10**exp.
func ratScale(x *big.Rat, exp int) {
	if exp < 0 {
		x.Inv(x)
		ratScale(x, -exp)
		x.Inv(x)
		return
	}
	for exp >= 9 {
		x.Quo(x, bigRatBillion)
		exp -= 9
	}
	for exp >= 1 {
		x.Quo(x, bigRatTen)
		exp--
	}
}

func (r BigRat) Eval(Context) Value {
	return r
}

func (r BigRat) toType(which valueType) Value {
	switch which {
	case intType:
		panic("big rat to int")
	case bigIntType:
		panic("big rat to big int")
	case bigRatType:
		return r
	case vectorType:
		return NewVector([]Value{r})
	case matrixType:
		return newMatrix([]Value{one, one}, []Value{r})
	}
	panic("BigRat.toType")
}

// shrink pulls, if possible, a BigRat down to a BigInt or Int.
func (r BigRat) shrink() Value {
	if !r.IsInt() {
		return r
	}
	return BigInt{r.Num()}.shrink()
}
