package httprouter

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_resolveKeyPairFromPattern(t *testing.T) {
	type args struct {
		pattern string
	}
	tests := []struct {
		name   string
		args   args
		wantKp []keyPair
	}{
		// TODO: Add test cases.
		{"test-0", args{"/a/b/c/d"}, nil},
		{"test-1", args{"/a/:name/c/:id"}, []keyPair{{2, "name"}, {4, "id"}}},
		{"test-2", args{"/a/b/:user/*id"}, []keyPair{{3, "user"}, {4, "id"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotKp := resolveKeyPairFromPattern(tt.args.pattern); !reflect.DeepEqual(gotKp, tt.wantKp) {
				t.Errorf("resolveKeyPairFromPattern() = %v, want %v", gotKp, tt.wantKp)
			}
		})
	}
}

func Test_resolveParamsFromPath(t *testing.T) {
	type args struct {
		path        string
		kp          []keyPair
		iswildChild bool
	}
	paramsPools.update(16)
	tests := []struct {
		name string
		args args
		want Params
	}{
		// TODO: Add test cases.
		{"test-0", args{"/a/b/c", nil, false}, nil},
		{"test-1", args{"/a/b/c", nil, true}, nil},
		{"test-2", args{"/a/tanzy2018/c/123", resolveKeyPairFromPattern("/a/:name/c/:id"), false},
			Params{{Key: "name", Value: "tanzy2018"}, {Key: "id", Value: "123"}}},
		{"test-3", args{"/a/tanzy2018/c/123", resolveKeyPairFromPattern("/a/:user/c/*id"), true},
			Params{{Key: "user", Value: "tanzy2018"}, {Key: "id", Value: "123"}}},
		{"test-4", args{"/a/tanzy2018/c", resolveKeyPairFromPattern("/a/:user/c/*id"), true},
			Params{{Key: "user", Value: "tanzy2018"}, {Key: "id", Value: ""}}},
		{"test-5", args{"/a/tanzy2018/c/d/e", resolveKeyPairFromPattern("/a/:user/c/*id"), true},
			Params{{Key: "user", Value: "tanzy2018"}, {Key: "id", Value: "d/e"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveParamsFromPath(tt.args.path, tt.args.kp, tt.args.iswildChild); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolveParamsFromPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unifyPattern(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{"test-0", args{"/a/b/c"}, "/a/b/c"},
		{"test-1", args{"/a/:name/c"}, fmt.Sprintf("/a/%s/c", placeHolder)},
		{"test-2", args{"/a/:user/c"}, fmt.Sprintf("/a/%s/c", placeHolder)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unifyPattern(tt.args.path); got != tt.want {
				t.Errorf("unifyPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_makeSegments(t *testing.T) {
	type args struct {
		path string
		max  int
	}
	tests := []struct {
		name          string
		args          args
		wantSegaments []string
	}{
		// TODO: Add test cases.
		{"test-0", args{"/a/b/c", 16}, []string{"a", "b", "c"}},
		{"test-1", args{"a/b/c", 16}, []string{"a", "b", "c"}},
		{"test-2", args{"a/b/c/", 16}, []string{"a", "b", "c"}},
		{"test-3", args{"a/b/c/", 2}, []string{"a", "b/c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotSegaments := makeSegments(tt.args.path, tt.args.max); !reflect.DeepEqual(gotSegaments, tt.wantSegaments) {
				t.Errorf("makeSegments() = %v, want %v", gotSegaments, tt.wantSegaments)
			}
		})
	}
}
