package main

import (
	"strconv"
)

type field struct {
	offset int64
	f      int
}

type fields []field

func (f fields) Len() int {
	return len(f)
}

func (f fields) Less(i, j int) bool {
	return f[i].f < f[j].f
}

func (f fields) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func newField(off int64, l string, i int) (field, error) {
	v, err := strconv.Atoi(l)
	if err != nil {
		return field{}, err
	}
	return field{off, v}, nil
}
