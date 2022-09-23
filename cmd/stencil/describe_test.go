// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the describe command

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_cleanPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "should support relative path",
			args: args{
				path: "foo/bar",
			},
			want:    "foo/bar",
			wantErr: false,
		},
		{
			name: "should support relative path with .",
			args: args{
				path: "./foo/bar",
			},
			want:    "foo/bar",
			wantErr: false,
		},
		{
			name: "should support relative path with ..",
			args: args{
				path: "../stencil/foo/bar",
			},
			want:    "foo/bar",
			wantErr: false,
		},
		{
			name: "should support absolute path",
			args: args{
				path: filepath.Join(cwd, "foo", "bar"),
			},
			want:    "foo/bar",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cleanPath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("cleanPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("cleanPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
