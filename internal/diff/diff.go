package diff

import (
	"github.com/google/go-cmp/cmp"
	"github.com/imdario/mergo"
)

func NotEmpty(expected, actual interface{}) (string, error) {
	if err := mergo.Merge(expected, actual, mergo.WithOverrideEmptySlice); err != nil {
		return "", err
	}

	return cmp.Diff(expected, actual), nil
}
