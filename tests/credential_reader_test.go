package tests

import (
	"context"
	"testing"

	vsphere "github.com/openshift/machine-api-operator/pkg/controller/vsphere"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestReadComponentCredentials verifies that the Machine API Operator
// reads vsphere-machine-api-creds from the openshift-machine-api namespace
func TestReadComponentCredentials(t *testing.T) {
	vcenterFQDN := "vcenter1.example.com"
	expectedUsername := "machine-api@vsphere.local"
	expectedPassword := "test-password"

	// Create test secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vsphere.ComponentCredentialSecretName,
			Namespace: vsphere.ComponentCredentialNamespace,
		},
		Data: map[string][]byte{
			vcenterFQDN + ".username": []byte(expectedUsername),
			vcenterFQDN + ".password": []byte(expectedPassword),
		},
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret).
		Build()

	credReader := vsphere.NewCredentialReader(fakeClient)

	cred, err := credReader.GetCredentialsForVCenter(context.Background(), vcenterFQDN)
	if err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	if cred.Username != expectedUsername {
		t.Errorf("Expected username %s, got %s", expectedUsername, cred.Username)
	}

	if cred.Password != expectedPassword {
		t.Errorf("Expected password %s, got %s", expectedPassword, cred.Password)
	}

	if cred.Server != vcenterFQDN {
		t.Errorf("Expected server %s, got %s", vcenterFQDN, cred.Server)
	}
}

// TestComponentCredentialFormat validates the expected secret format
func TestComponentCredentialFormat(t *testing.T) {
	tests := []struct {
		name    string
		secret  *corev1.Secret
		wantErr bool
	}{
		{
			name: "Valid single vCenter credential",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
				Data: map[string][]byte{
					"vcenter1.example.com.username": []byte("user"),
					"vcenter1.example.com.password": []byte("pass"),
				},
			},
			wantErr: false,
		},
		{
			name: "Valid multiple vCenter credentials",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
				Data: map[string][]byte{
					"vcenter1.example.com.username": []byte("user1"),
					"vcenter1.example.com.password": []byte("pass1"),
					"vcenter2.example.com.username": []byte("user2"),
					"vcenter2.example.com.password": []byte("pass2"),
				},
			},
			wantErr: false,
		},
		{
			name: "Missing password",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
				Data: map[string][]byte{
					"vcenter1.example.com.username": []byte("user"),
				},
			},
			wantErr: true,
		},
		{
			name: "Empty secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
				Data:       map[string][]byte{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vsphere.ValidateSecretFormat(tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSecretFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCredentialFallback verifies fallback to shared credentials
func TestCredentialFallback(t *testing.T) {
	vcenterFQDN := "vcenter1.example.com"
	expectedUsername := "shared-user@vsphere.local"
	expectedPassword := "shared-password"

	// Create shared credential secret (no component secret)
	sharedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vsphere.SharedCredentialSecretName,
			Namespace: vsphere.SharedCredentialNamespace,
		},
		Data: map[string][]byte{
			vcenterFQDN + ".username": []byte(expectedUsername),
			vcenterFQDN + ".password": []byte(expectedPassword),
		},
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sharedSecret).
		Build()

	credReader := vsphere.NewCredentialReader(fakeClient)

	// Should fall back to shared credentials
	cred, err := credReader.GetCredentialsForVCenter(context.Background(), vcenterFQDN)
	if err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	if cred.Username != expectedUsername {
		t.Errorf("Expected username %s, got %s", expectedUsername, cred.Username)
	}
}
