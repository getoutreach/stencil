// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the stencil package

package stencil_test

import (
	"fmt"
	"time"

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

	fmt.Println(l.Generated.UTC().Format(time.RFC3339Nano))

	// Output:
	// 2022-04-01T00:25:51.047307Z
}
