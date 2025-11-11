package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	userpb "go-monorepo-template/proto/acme/user/v1"
)

func TestGetUser(t *testing.T) {
	s := &server{}

	testCases := []struct {
		name        string
		req         *userpb.GetUserRequest
		expectedErr bool
	}{
		{
			name: "valid request",
			req:  &userpb.GetUserRequest{UserId: "123"},
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.GetUser(context.Background(), tc.req)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
