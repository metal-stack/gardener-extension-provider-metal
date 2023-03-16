package mutator

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_shoot_Mutate(t *testing.T) {
	type fields struct {
		client  client.Client
		decoder runtime.Decoder
	}
	type args struct {
		ctx context.Context
		new client.Object
		old client.Object
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &mutator{
				client:  tt.fields.client,
				decoder: tt.fields.decoder,
			}
			if err := s.Mutate(tt.args.ctx, tt.args.new, tt.args.old); (err != nil) != tt.wantErr {
				t.Errorf("shoot.Mutate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
