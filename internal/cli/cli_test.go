package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	for _, test := range []struct {
		args           []string
		command, shell string
		fail           bool
	}{
		{nil, "help", "", false},
		{[]string{"--version"}, "version", "", false},
		{[]string{"-v"}, "version", "", false},
		{[]string{"auth"}, "login", "", false},
		{[]string{"lock", "--shell=fish"}, "logout", "fish", false},
		{[]string{"hook", "zsh"}, "hook", "zsh", false},
		{[]string{"init", "--help"}, "help", "", false},
		{[]string{"export", "--provider"}, "", "", true},
		{[]string{"export", "--unknown"}, "", "", true},
		{[]string{"login", "extra"}, "", "", true},
		{[]string{"hook", "bash", "extra"}, "", "", true},
		{[]string{"typo"}, "", "", true},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			request, err := Parse(test.args)
			if (err != nil) != test.fail {
				t.Fatalf("Parse error = %v, want failure %v", err, test.fail)
			}
			if !test.fail && (request.Command != test.command || request.Shell != test.shell) {
				t.Fatalf("unexpected request: %+v", request)
			}
		})
	}
	request, err := Parse([]string{"load", "--provider=bitwarden", "--folder", "Folder with spaces", "--folder-id=id", "--items", "one, two,,", "--quiet"})
	if err != nil || request.Command != "export" || request.Folder != "Folder with spaces" || !request.Quiet || !reflect.DeepEqual(request.Items, []string{"one", "two"}) {
		t.Fatalf("export request: %+v, %v", request, err)
	}
}
