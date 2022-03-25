// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements a dotnotation parser for
// accessing a map[string]interface{}

// Package dotnotation implements a dotnotation (hello.world) for
// accessing fields within a map[string]interface{}
package dotnotation

import (
	"reflect"
	"testing"
)

func TestGet(t *testing.T) {
	type args struct {
		data map[string]interface{}
		key  string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "should handle basic depths",
			args: args{
				key: "hello.world",
				data: map[string]interface{}{
					"hello": map[string]interface{}{
						"world": "hello, world!",
					},
				},
			},
			want:    "hello, world!",
			wantErr: false,
		},
		{
			name: "should fail on invalid keys",
			args: args{
				key:  "hello.world",
				data: map[string]interface{}{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Get(tt.args.data, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_get(t *testing.T) {
	type args struct {
		data map[string]interface{}
		key  string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := get(tt.args.data, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("get() = %v, want %v", got, tt.want)
			}
		})
	}
}
