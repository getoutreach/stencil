// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the public API for templates
// for stencil

package codegen

import (
	"reflect"
	"testing"
	"time"

	"github.com/getoutreach/stencil/internal/modules/modulestest"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5"
	"github.com/sirupsen/logrus"
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

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestTplStencil_GetModuleHook(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		inserts []interface{}
		args    args
		want    []interface{}
	}{
		{
			inserts: []interface{}{
				[]string{"abc"},
				[]string{"def"},
				[]interface{}{map[string]interface{}{
					"abc": "def",
				}},
				[]string{"abc"},
			},
			args: args{
				name: "name",
			},
			want: []interface{}{
				// This is what the hashing resulted in
				map[string]interface{}{
					"abc": "def",
				},
				"def",
				"abc",
				"abc",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &TplStencil{
				t: must(
					NewTemplate(
						must(modulestest.NewModuleFromTemplates(&configuration.TemplateRepositoryManifest{
							Name: "test",
						})),
						"not-a-real-template.tpl",
						0o755,
						time.Now(),
						[]byte(""),
						logrus.New(),
					),
				),
				s: &Stencil{sharedData: &sharedData{
					moduleHooks: make(map[string][]interface{}),
					globals:     make(map[string]global),
				}},
				log: logrus.New(),
			}

			s.s.isFirstPass = true
			for _, insert := range tt.inserts {
				err1, err2 := s.AddToModuleHook(s.t.Module.Name, tt.args.name, insert)
				if err1 != nil || err2 != nil {
					t.Errorf("TplStencil.GetModuleHook() error = %v, %v", err1, err2)
					return
				}
			}
			s.s.isFirstPass = false

			if got := s.GetModuleHook(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TplStencil.GetModuleHook() = %v, want %v", got, tt.want)
			}
		})
	}
}
