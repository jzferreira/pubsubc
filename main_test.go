package main

import (
	"reflect"
	"testing"
)

func Test_parseEnv(t *testing.T) {
	type args struct {
		env string
	}
	type test struct {
		name  string
		args  args
		want  Topics
		want1 string
	}

	tests := []test {
		test {
			name : "single",
			args : args {
				env: "project1,topic1,topic2>subscription1",
			},
			want : Topics{"topic1": []string{}, "topic2": []string{"subscription1"} },
			want1 : "project1",
		},
		test {
			name : "with push configs",
			args : args {
				env: "project1,topic1,topic2>subscription1@http://localhost:3333",
			},
			want : Topics{"topic1": []string{}, "topic2": []string{"subscription1@http://localhost:3333"} },
			want1 : "project1",
		},

	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ParseEnv(tt.args.env)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseEnv() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ParseEnv() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}