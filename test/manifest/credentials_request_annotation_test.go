package manifest_test

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const vsphereComponentAnnotation = "cloudcredential.openshift.io/vsphere-component"

type credentialsRequest struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name        string            `yaml:"name"`
		Annotations map[string]string `yaml:"annotations"`
	} `yaml:"metadata"`
}

func parseCredentialsRequests(t *testing.T, path string) []credentialsRequest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out []credentialsRequest
	for _, doc := range strings.Split(string(data), "\n---") {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var cr credentialsRequest
		if err := yaml.Unmarshal([]byte(doc), &cr); err != nil {
			t.Fatalf("unmarshal %s: %v", path, err)
		}
		if cr.Kind == "CredentialsRequest" {
			out = append(out, cr)
		}
	}
	return out
}

func findCR(crs []credentialsRequest, name string) (credentialsRequest, bool) {
	for _, cr := range crs {
		if cr.Metadata.Name == name {
			return cr, true
		}
	}
	return credentialsRequest{}, false
}

func TestMAOVSphereCredentialsRequestAnnotation(t *testing.T) {
	path := "../../install/0000_30_machine-api-operator_00_credentials-request.yaml"
	crs := parseCredentialsRequests(t, path)

	cr, ok := findCR(crs, "openshift-machine-api-vsphere")
	if !ok {
		t.Fatal("CredentialsRequest 'openshift-machine-api-vsphere' not found in manifest")
	}

	got, present := cr.Metadata.Annotations[vsphereComponentAnnotation]
	if !present {
		t.Fatalf("annotation %q missing from openshift-machine-api-vsphere", vsphereComponentAnnotation)
	}
	if got != "machineAPI" {
		t.Errorf("annotation value: got %q, want %q", got, "machineAPI")
	}
}

func TestNonVSphereCredentialsRequestsUnmodified(t *testing.T) {
	path := "../../install/0000_30_machine-api-operator_00_credentials-request.yaml"
	crs := parseCredentialsRequests(t, path)

	nonVSphere := []string{
		"openshift-machine-api-aws",
		"openshift-machine-api-azure",
		"openshift-machine-api-gcp",
		"openshift-machine-api-openstack",
	}
	for _, name := range nonVSphere {
		cr, ok := findCR(crs, name)
		if !ok {
			continue
		}
		if _, present := cr.Metadata.Annotations[vsphereComponentAnnotation]; present {
			t.Errorf("non-vsphere CR %q must NOT carry %s", name, vsphereComponentAnnotation)
		}
	}
}

func TestAnnotationValueIsNotEmpty(t *testing.T) {
	path := "../../install/0000_30_machine-api-operator_00_credentials-request.yaml"
	crs := parseCredentialsRequests(t, path)

	cr, ok := findCR(crs, "openshift-machine-api-vsphere")
	if !ok {
		t.Fatal("CredentialsRequest 'openshift-machine-api-vsphere' not found in manifest")
	}
	val := cr.Metadata.Annotations[vsphereComponentAnnotation]
	if strings.TrimSpace(val) == "" {
		t.Errorf("annotation %q must not be empty", vsphereComponentAnnotation)
	}
}
