package tests

import (
	"testing"

	vsphere "github.com/openshift/machine-api-operator/pkg/controller/vsphere"
)

// TestRequiredPrivilegesCount verifies that at least 35 privileges are defined
func TestRequiredPrivilegesCount(t *testing.T) {
	expectedMinCount := 35
	actualCount := len(vsphere.RequiredMachineAPIPrivileges)

	if actualCount < expectedMinCount {
		t.Errorf("Expected at least %d required privileges, got %d", expectedMinCount, actualCount)
	}

	t.Logf("Machine API requires %d vSphere privileges", actualCount)
}

// TestRequiredPrivilegesCategories verifies all privilege categories are present
func TestRequiredPrivilegesCategories(t *testing.T) {
	requiredCategories := map[string]bool{
		"VirtualMachine.Config":    false,
		"VirtualMachine.Interact":  false,
		"VirtualMachine.Inventory": false,
		"Resource.AssignVMToPool":  false,
		"Datastore.AllocateSpace":  false,
		"Datastore.FileManagement": false,
		"Network.Assign":           false,
	}

	for _, priv := range vsphere.RequiredMachineAPIPrivileges {
		for category := range requiredCategories {
			if hasPrefix(priv, category) {
				requiredCategories[category] = true
			}
		}
	}

	for category, found := range requiredCategories {
		if !found {
			t.Errorf("Required privilege category %s not found in privilege list", category)
		}
	}
}

// TestFormatMissingPrivilegesError verifies error formatting
func TestFormatMissingPrivilegesError(t *testing.T) {
	vcenter := "vcenter1.example.com"
	missing := []string{
		"VirtualMachine.Config.AddNewDisk",
		"VirtualMachine.Inventory.Create",
	}

	err := vsphere.FormatMissingPrivilegesError(vcenter, missing)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	errMsg := err.Error()
	if len(errMsg) == 0 {
		t.Error("Error message is empty")
	}

	// Verify error message contains vcenter and privilege count
	expectedCount := "2"
	if !contains(errMsg, vcenter) {
		t.Errorf("Error message does not contain vCenter name: %s", errMsg)
	}
	if !contains(errMsg, expectedCount) {
		t.Errorf("Error message does not contain privilege count: %s", errMsg)
	}
}

// TestPrivilegeValidationStructure verifies validation result structure
func TestPrivilegeValidationStructure(t *testing.T) {
	// Test that ValidationResult can be created and has expected fields
	result := &vsphere.ValidationResult{
		Valid:            false,
		MissingPrivileges: []string{"VirtualMachine.Config.AddNewDisk"},
		ValidationErrors:  []error{},
	}

	if result.Valid {
		t.Error("Expected Valid to be false")
	}

	if len(result.MissingPrivileges) != 1 {
		t.Errorf("Expected 1 missing privilege, got %d", len(result.MissingPrivileges))
	}
}

// hasPrefix checks if string s starts with prefix
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// contains checks if string s contains substr
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
