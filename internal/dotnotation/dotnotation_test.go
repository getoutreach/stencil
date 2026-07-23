// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the dotnotation parser.

package dotnotation

import (
	"reflect"
	"testing"
)

func TestGet(t *testing.T) {
	type args struct {
		data any
		key  string
	}
	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "should handle basic depths",
			args: args{
				key: "hello.world",
				data: map[any]any{
					"hello": map[any]any{
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
				data: map[any]any{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should support map[int]",
			args: args{
				key: "1.2.3",
				data: map[int]any{
					1: map[int]any{
						2: map[int]any{
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
				data: map[string]any{
					"1": map[int]any{
						2: map[int]any{
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
