// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the public API for templates
// for stencil

package codegen

import (
	"context"
	"reflect"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/internal/modules/modulestest"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/sirupsen/logrus"
)

type testTpl struct {
	s   *Stencil
	t   *Template
	log logrus.FieldLogger
}

// fakeTemplate returns a faked struct suitable for testing
// template functions
func fakeTemplate(t *testing.T, args map[string]interface{},
	requestArgs map[string]configuration.Argument) *testTpl {
	test := &testTpl{}
	log := logrus.New()

	m, err := modulestest.NewModuleFromTemplates(requestArgs, "test")
	if err != nil {
		t.Fatal(err)
	}

	fs, err := m.GetFS(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f, err := fs.Create("templates/test.tpl")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	test.s = NewStencil(&configuration.ServiceManifest{
		Name:      "testing",
		Arguments: args,
		Modules:   []*configuration.TemplateRepository{{Name: m.Name}},
	}, []*modules.Module{m}, log)

	// use the first template from the module
	// which we've created earlier after loading the module in the
	// NewModuleFromTemplates call. This won't be used, but it's
	// enough to set up the correct environment for running template test functions.
	tpls, err := test.s.getTemplates(context.Background(), log)
	if err != nil {
		t.Fatal(err)
	}
	test.t = tpls[0]

	return test
}

func TestTplStencil_Arg(t *testing.T) {
	type args struct {
		pth string
	}
	tests := []struct {
		name    string
		fields  *testTpl
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "should support basic argument",
			fields: fakeTemplate(t, map[string]interface{}{
				"hello": "world",
			}, map[string]configuration.Argument{
				"hello": {},
			}),
			args: args{
				pth: "hello",
			},
			want:    "world",
			wantErr: false,
		},
		{
			name: "should fail when an argument is not defined",
			fields: fakeTemplate(t, map[string]interface{}{
				"hello": "world",
			}, map[string]configuration.Argument{}),
			args: args{
				pth: "hello",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "should support basic JSON schema",
			fields: fakeTemplate(t, map[string]interface{}{
				"hello": "world",
			}, map[string]configuration.Argument{
				"hello": {
					Schema: map[string]interface{}{
						"type": "string",
					},
				},
			}),
			args: args{
				pth: "hello",
			},
			want:    "world",
			wantErr: false,
		},
		{
			name: "should fail when provided value doesn't match json schema",
			fields: fakeTemplate(t, map[string]interface{}{
				"hello": 1,
			}, map[string]configuration.Argument{
				"hello": {
					Schema: map[string]interface{}{
						"type": "string",
					},
				},
			}),
			args: args{
				pth: "hello",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should support nested json schema",
			fields: fakeTemplate(t, map[string]interface{}{
				"hello": map[string]interface{}{
					"world": map[string]interface{}{
						"abc": []string{"def"},
					},
				},
			}, map[string]configuration.Argument{
				"hello": {
					Schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"world": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"abc": map[string]interface{}{
										"type": "array",
									},
								},
							},
						},
					},
				},
			}),
			args: args{
				pth: "hello",
			},
			want:    map[string]interface{}{"world": map[string]interface{}{"abc": []string{"def"}}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &TplStencil{
				s:   tt.fields.s,
				t:   tt.fields.t,
				log: tt.fields.log,
			}
			got, err := s.Arg(tt.args.pth)
			if (err != nil) != tt.wantErr {
				t.Errorf("TplStencil.Arg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TplStencil.Arg() = %v, want %v", got, tt.want)
			}
		})
	}
}
