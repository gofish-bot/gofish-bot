package generic

import (
	"fmt"
	"testing"
)

func Test_getVersion(t *testing.T) {
	type args struct {
		releaseName string
		appName     string
	}
	tests := []struct {
		releaseName string
		appName     string
		want        string
	}{
		{releaseName: "1.0.0", appName: "myapp", want: "1.0.0"},
		{releaseName: "v1.0.0", appName: "myapp", want: "1.0.0"},
		{releaseName: "myapp-1.0.0", appName: "myapp", want: "1.0.0"},
		{releaseName: "myapp/1.0.0", appName: "myapp", want: "1.0.0"},
		{releaseName: "myapp/v1.0.0", appName: "myapp", want: "1.0.0"},
		{releaseName: "myapp1.0.0", appName: "myapp", want: "1.0.0"},
		{releaseName: "myappv1.0.0", appName: "myapp", want: "1.0.0"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Relesse: '%s' for %s gives clean version %s", tt.releaseName, tt.appName, tt.want), func(t *testing.T) {
			if got := getVersion(tt.releaseName, tt.appName); got != tt.want {
				t.Errorf("getVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
