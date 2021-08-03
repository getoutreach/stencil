package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/getoutreach/stencil/internal/tests"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/sirupsen/logrus"
)

func main() {
	generators := []manifestGenerator{}

	for _, generator := range generators {
		m := json.NewEncoder(os.Stdout)
		for {
			name, manifest := generator.Next()
			if manifest == nil {
				break
			}

			t, err := tests.NewTest(name, manifest)
			if err != nil {
				logrus.WithError(err).Fatal("failed to encode test")
			}

			err = m.Encode(t)
			if err != nil {
				logrus.WithError(err).Fatal("failed to serialize test")
			}
		}
	}
}

type manifestOption func(*configuration.ServiceManifest, int) string

type manifestGenerator struct {
	options [][]manifestOption
	indices []int
	done    bool
}

// Next gets the next available manifest and the friendly name
// associated with this variation.
//
// Options are a two dimensional array.  The outer dimension is a
// collection of independent options/variations.  The inner dimension
// is variations of a particular value (such as true & false for a
// boolean flag).
//
// The algorithm works by maintaining the index for each
// variation. After a specific run, it attempts to pick the next
// version left-to-right. When it finds an element which is not at its
// last variation, it increments this index zeroing everything before
// (so the run will iterate through all their variations).
func (m *manifestGenerator) Next() (string, *configuration.ServiceManifest) {
	if m.done {
		return "", nil
	}

	names := make([]string, len(m.options))
	incremented := false
	manifest := configuration.ServiceManifest{Name: "mysvc"}
	for kk, opt := range m.options {
		names[kk] = opt[m.indices[kk]](&manifest, m.indices[kk])
		if incremented || m.indices[kk]+1 == len(opt) {
			continue
		}
		incremented = true
		m.indices[kk]++
		for jj := 0; jj < kk; jj++ {
			m.indices[jj] = 0
		}
	}

	m.done = m.done || !incremented
	return strings.Join(names, " "), &manifest
}
