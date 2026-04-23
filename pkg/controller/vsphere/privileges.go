/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vsphere

import (
	"context"
	"fmt"
	"sort"

	"github.com/vmware/govmomi/session/keepalive"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	"k8s.io/klog/v2"
)

// RequiredMachineAPIPrivileges defines the 35 vSphere privileges required
// for Machine API Operator operations according to Epic #14 design
var RequiredMachineAPIPrivileges = []string{
	// VirtualMachine.Config.* - VM configuration management
	"VirtualMachine.Config.AddExistingDisk",
	"VirtualMachine.Config.AddNewDisk",
	"VirtualMachine.Config.AddRemoveDevice",
	"VirtualMachine.Config.AdvancedConfig",
	"VirtualMachine.Config.Annotation",
	"VirtualMachine.Config.CPUCount",
	"VirtualMachine.Config.ChangeTracking",
	"VirtualMachine.Config.DiskExtend",
	"VirtualMachine.Config.DiskLease",
	"VirtualMachine.Config.EditDevice",
	"VirtualMachine.Config.Memory",
	"VirtualMachine.Config.MksControl",
	"VirtualMachine.Config.QueryFTCompatibility",
	"VirtualMachine.Config.QueryUnownedFiles",
	"VirtualMachine.Config.RawDevice",
	"VirtualMachine.Config.ReloadFromPath",
	"VirtualMachine.Config.RemoveDisk",
	"VirtualMachine.Config.Rename",
	"VirtualMachine.Config.ResetGuestInfo",
	"VirtualMachine.Config.Resource",
	"VirtualMachine.Config.Settings",
	"VirtualMachine.Config.SwapPlacement",
	"VirtualMachine.Config.UpgradeVirtualHardware",

	// VirtualMachine.Interact.* - Power and console operations
	"VirtualMachine.Interact.ConsoleInteract",
	"VirtualMachine.Interact.PowerOff",
	"VirtualMachine.Interact.PowerOn",
	"VirtualMachine.Interact.Reset",
	"VirtualMachine.Interact.Suspend",

	// VirtualMachine.Inventory.* - VM lifecycle management
	"VirtualMachine.Inventory.Create",
	"VirtualMachine.Inventory.Delete",
	"VirtualMachine.Inventory.Move",
	"VirtualMachine.Inventory.Register",
	"VirtualMachine.Inventory.Unregister",

	// Resource and storage privileges
	"Resource.AssignVMToPool",

	// Datastore privileges
	"Datastore.AllocateSpace",
	"Datastore.FileManagement",

	// Network privileges
	"Network.Assign",
}

// PrivilegeValidator validates vSphere privileges for a given credential
type PrivilegeValidator struct {
	client *vim25.Client
}

// NewPrivilegeValidator creates a new PrivilegeValidator
func NewPrivilegeValidator(client *vim25.Client) *PrivilegeValidator {
	return &PrivilegeValidator{
		client: client,
	}
}

// ValidationResult contains the result of privilege validation
type ValidationResult struct {
	Valid            bool
	MissingPrivileges []string
	ValidationErrors  []error
}

// ValidateMachineAPIPrivileges checks if the authenticated user has all required privileges
func (pv *PrivilegeValidator) ValidateMachineAPIPrivileges(ctx context.Context) (*ValidationResult, error) {
	klog.V(4).Info("Validating Machine API privileges")

	result := &ValidationResult{
		Valid:            true,
		MissingPrivileges: []string{},
		ValidationErrors:  []error{},
	}

	// Get session manager to check current user
	sessionManager := pv.client.ServiceContent.SessionManager
	if sessionManager == nil {
		return nil, fmt.Errorf("session manager not available")
	}

	// Get current session to identify the user
	currentSession, err := methods.GetCurrentSession(ctx, pv.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get current session: %w", err)
	}

	if currentSession == nil || currentSession.UserName == "" {
		return nil, fmt.Errorf("no active session found")
	}

	klog.V(4).Infof("Checking privileges for user: %s", currentSession.UserName)

	// Get authorization manager
	authManager := pv.client.ServiceContent.AuthorizationManager
	if authManager == nil {
		return nil, fmt.Errorf("authorization manager not available")
	}

	// Check each required privilege
	// Note: In a real implementation, we would check privileges on specific managed objects
	// For now, we'll validate that the user has these privileges assigned to their role
	hasPrivilegeReq := types.HasPrivilegeOnEntities{
		This:      *authManager,
		Entity:    []types.ManagedObjectReference{pv.client.ServiceContent.RootFolder},
		SessionId: currentSession.Key,
		PrivId:    RequiredMachineAPIPrivileges,
	}

	hasPrivilegeResp, err := methods.HasPrivilegeOnEntities(ctx, pv.client, &hasPrivilegeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to check privileges: %w", err)
	}

	// Process results
	if len(hasPrivilegeResp.Returnval) == 0 {
		return nil, fmt.Errorf("no privilege check results returned")
	}

	// hasPrivilegeResp.Returnval contains one EntityPrivilege per entity checked
	entityPrivileges := hasPrivilegeResp.Returnval[0]

	// Build a map of privileges we have
	privilegeMap := make(map[string]bool)
	for _, privCheck := range entityPrivileges.PrivAvailability {
		privilegeMap[privCheck.PrivId] = privCheck.IsGranted
	}

	// Check for missing privileges
	for _, requiredPriv := range RequiredMachineAPIPrivileges {
		if !privilegeMap[requiredPriv] {
			result.MissingPrivileges = append(result.MissingPrivileges, requiredPriv)
			result.Valid = false
		}
	}

	// Sort missing privileges for consistent output
	sort.Strings(result.MissingPrivileges)

	if len(result.MissingPrivileges) > 0 {
		klog.Warningf("Missing %d required privileges: %v", len(result.MissingPrivileges), result.MissingPrivileges)
	} else {
		klog.V(4).Info("All required Machine API privileges validated successfully")
	}

	return result, nil
}

// FormatMissingPrivilegesError creates a detailed error message for missing privileges
func FormatMissingPrivilegesError(vcenter string, missing []string) error {
	return fmt.Errorf("insufficient privileges for vCenter %s: missing %d privileges: %v",
		vcenter, len(missing), missing)
}

// ValidatePrivilegesWithRetry validates privileges with automatic retry on transient errors
func (pv *PrivilegeValidator) ValidatePrivilegesWithRetry(ctx context.Context, maxRetries int) (*ValidationResult, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		result, err := pv.ValidateMachineAPIPrivileges(ctx)
		if err == nil {
			return result, nil
		}

		// Check if error is retryable (e.g., network timeout, session expired)
		if !isRetryableError(err) {
			return nil, err
		}

		klog.V(4).Infof("Privilege validation attempt %d/%d failed: %v", i+1, maxRetries, err)
		lastErr = err

		// Re-establish keepalive if session was lost
		if pv.client.Client != nil {
			_ = keepalive.Start(ctx, pv.client.Client, 1)
		}
	}

	return nil, fmt.Errorf("privilege validation failed after %d retries: %w", maxRetries, lastErr)
}

// isRetryableError determines if an error is transient and can be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Common transient errors
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"session is not authenticated",
		"session has expired",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains is a simple substring check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexContains(s, substr)))
}

func indexContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
