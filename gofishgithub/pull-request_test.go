package gofishgithub

import "testing"

func Test_cleanReleaseDescription(t *testing.T) {
	type args struct {
		releaseDescription string
	}
	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name: "Does not hurt headers",
			markdown: `# Header 1
			## Header 2
			### Header 1`,
			want: `# Header 1
			## Header 2
			### Header 1`,
		},
		{
			name: "Replaces issue numbers",
			markdown: `# Header 1
			Fixes issue #123 which was no good...!`,
			want: `# Header 1
			Fixes issue #<!-- -->123 which was no good...!`,
		},
		{
			name: "Replaces mentions",
			markdown: `# Header 1
			@gofish-bot does stuff! @GoFiSh-BOT`,
			want: `# Header 1
			@<!-- -->gofish-bot does stuff! @<!-- -->GoFiSh-BOT`,
		},
		{
			name: "Replaces links",
			markdown: `# Header 1
			[Here i am](https://github.com/gofish-bot/gofish-bot/) `,
			want: `# Header 1
			https:<span/>/<span/>/github<span/>.com<span/>/gofish-bot<span/>/gofish-bot<span/>/ `,
		},
		{
			name: "Replaces links without markdown syntax",
			markdown: `# Header 1
			Github does link magic.. https://github.com/gofish-bot/gofish-bot/ `,
			want: `# Header 1
			Github does link magic.. https:<span/>/<span/>/github<span/>.com<span/>/gofish-bot<span/>/gofish-bot<span/>/ `,
		},
		// We may want to add this?
		// {
		// 	name: "Replaces github repo links",
		// 	markdown: `fluxcd/flux-cli:v0.12.1`,
		// 	want: `fluxcd<span/>/flux-cli:v0.12.1`,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanMarkdown(tt.markdown); got != tt.want {
				t.Errorf("cleanReleaseDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}
