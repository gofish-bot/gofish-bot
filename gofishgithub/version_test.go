package gofishgithub

import (
	"context"
	"testing"

	"github.com/gofish-bot/gofish-bot/models"
	"github.com/google/go-github/v32/github"
	ghApi "github.com/google/go-github/v32/github"
)

func TestGoFish_getVersion(t *testing.T) {
	type fields struct {
		Client      *ghApi.Client
		BotOrg      string
		FoodRepo    string
		FoodOrg     string
		AuthorName  string
		AuthorEmail string
	}
	type args struct {
		ctx context.Context
		app models.DesiredApp
		ref string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test 1",
			fields: fields{
				Client:      github.NewClient(nil),
				BotOrg:      "gofish-bot",
				FoodRepo:    "fish-food",
				FoodOrg:     "fishworks",
				AuthorName:  "none",
				AuthorEmail: "no@ne",
			},
			args: args{
				ctx: context.Background(),
				app: models.DesiredApp{
					Name: "glooctl",
					Repo: "glooctl",
				},
				ref: "3e06512e56f416a6a8e154174012d1422f2b5f92",
			},
			want:    "0.17.2",
			wantErr: false,
		},
		{
			name: "test 2",
			fields: fields{
				Client:      github.NewClient(nil),
				BotOrg:      "gofish-bot",
				FoodRepo:    "fish-food",
				FoodOrg:     "fishworks",
				AuthorName:  "none",
				AuthorEmail: "no@ne",
			},
			args: args{

				ctx: context.Background(),
				app: models.DesiredApp{
					Name: "glooctl",
					Repo: "glooctl",
				},
				ref: "60d5d37507c11390d1d57812663c57f5432b314f",
			},
			want:    "0.17.4",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &GoFish{
				Client:      tt.fields.Client,
				BotOrg:      tt.fields.BotOrg,
				FoodRepo:    tt.fields.FoodRepo,
				FoodOrg:     tt.fields.FoodOrg,
				AuthorName:  tt.fields.AuthorName,
				AuthorEmail: tt.fields.AuthorEmail,
			}
			got, err := p.getVersion(tt.args.ctx, tt.args.app, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GoFish.getVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GoFish.getVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
