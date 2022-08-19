// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the stencil package

package stencil_test

import (
	"fmt"

	"github.com/getoutreach/stencil/pkg/stencil"
)

func ExampleLoadLockfile() {
	// Load the lockfile
	l, err := stencil.LoadLockfile("testdata")
	if err != nil {
		// handle the error
		fmt.Println(err)
		return
	}

	fmt.Println(l.Version)

	// Output:
	// v1.6.2
}
