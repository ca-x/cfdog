package ghrelase

import "testing"

func TestGetLatest(t *testing.T) {
	type args struct {
		repoName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "hugo", args: args{repoName: "gohugoio/hugo"}, want: "v0.134.3", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLatest(tt.args.repoName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetLatest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
