// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the public API for templates
// for stencil

package codegen

import (
	"reflect"
	"testing"

	"github.com/go-git/go-billy/v5"
)

func TestTplStencil_ReadBlocks(t *testing.T) {
	type fields struct {
	}
	type args struct {
		fpath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]string
		wantErr error
	}{
		{
			name: "should read blocks from a file",
			args: args{
				fpath: "testdata/blocks-test.txt",
			},
			want: map[string]string{
				"helloWorld": "Hello, world!",
				"e2e":        "content",
			},
		},
		{
			name: "should error on out of chroot path",
			args: args{
				fpath: "../testdata/blocks-test.txt",
			},
			wantErr: billy.ErrCrossedBoundary,
		},
		{
			name: "should return no data on non-existent file",
			args: args{
				fpath: "testdata/does-not-exist.txt",
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &TplStencil{}
			got, err := s.ReadBlocks(tt.args.fpath)

			// String checking because errors.Is isn't working
			if (tt.wantErr != nil) && err.Error() != tt.wantErr.Error() {
				t.Errorf("TplStencil.ReadBlocks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TplStencil.ReadBlocks() = %v, want %v", got, tt.want)
			}
		})
	}
}
