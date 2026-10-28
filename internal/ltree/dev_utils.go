package ltree

import "fmt"

// Debug функции

func printPointersSlice[T any](S []*T) {
	if S == nil || len(S) == 0 {
		fmt.Println("<nil slice>")
		return
	}

	for i, s := range S {
		if i > 0 {
			fmt.Print(" ")
		}

		if s == nil {
			fmt.Print("<nil elem>; ")
		} else {
			fmt.Printf("%+v; ", *s)
		}
	}
	fmt.Println()
}

func printSlice[T any](S []T) {
	if S == nil || len(S) == 0 {
		fmt.Println("<nil slice>; ")
		return
	}

	for i, s := range S {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%+v; ", s)
	}
	fmt.Println()
}

func printSliceOfPointersSlice[T any](S [][]*T) {
	if S == nil {
		fmt.Print("<nil slice of ptr slice>")
	}
	for _, s := range S {
		printPointersSlice(s)
	}
}
