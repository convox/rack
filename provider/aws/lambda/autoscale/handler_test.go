package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudformation"
)

func TestShouldAutoscale(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{cloudformation.StackStatusCreateComplete, true},
		{cloudformation.StackStatusUpdateComplete, true},
		{cloudformation.StackStatusCreateInProgress, false},
		{cloudformation.StackStatusCreateFailed, false},
		{cloudformation.StackStatusRollbackInProgress, false},
		{cloudformation.StackStatusRollbackFailed, false},
		{cloudformation.StackStatusRollbackComplete, false},
		{cloudformation.StackStatusDeleteInProgress, false},
		{cloudformation.StackStatusDeleteFailed, false},
		{cloudformation.StackStatusDeleteComplete, false},
		{cloudformation.StackStatusUpdateInProgress, false},
		{cloudformation.StackStatusUpdateFailed, false},
		{cloudformation.StackStatusUpdateCompleteCleanupInProgress, false},
		{cloudformation.StackStatusUpdateRollbackInProgress, false},
		{cloudformation.StackStatusUpdateRollbackFailed, false},
		{cloudformation.StackStatusUpdateRollbackCompleteCleanupInProgress, false},
		{cloudformation.StackStatusUpdateRollbackComplete, false},
		{cloudformation.StackStatusReviewInProgress, false},
		{cloudformation.StackStatusImportInProgress, false},
		{cloudformation.StackStatusImportComplete, false},
		{cloudformation.StackStatusImportRollbackInProgress, false},
		{cloudformation.StackStatusImportRollbackFailed, false},
		{cloudformation.StackStatusImportRollbackComplete, false},
		{"", false},
		{"UNKNOWN_STATUS", false},
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			got := shouldAutoscale(tc.status)
			if got != tc.want {
				t.Errorf("shouldAutoscale(%q) = %v; want %v", tc.status, got, tc.want)
			}
		})
	}
}
