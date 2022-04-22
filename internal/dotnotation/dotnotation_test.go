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
		data interface{}
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
				data: map[interface{}]interface{}{
					"hello": map[interface{}]interface{}{
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
				data: map[interface{}]interface{}{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should support map[int]",
			args: args{
				key: "1.2.3",
				data: map[int]interface{}{
					1: map[int]interface{}{
						2: map[int]interface{}{
							3: 4,
						},
					},
				},
			},
			want:    4,
			wantErr: false,
		},
		{
			name: "should support nested maps",
			args: args{
				key: "1.2.3",
				data: map[string]interface{}{
					"1": map[int]interface{}{
						2: map[int]interface{}{
							3: 4,
						},
					},
				},
			},
			want:    4,
			wantErr: false,
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
