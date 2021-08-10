package main

import (
	"errors"
	"fmt"
)

type errs []error

func (es errs) Error() string {
	switch len(es) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%v", es[0])
	}

	rt := "composite error:"
	for _, e := range es {
		if e != nil {
			rt = fmt.Sprintf("%s\n\t%v", rt, e)
		}
	}
	return rt
}

func (es errs) Is(target error) bool {
	if len(es) == 0 {
		return false
	}

	for _, e := range es {
		if errors.Is(e, target) {
			return true
		}
	}
	return false
}
