package native

import (
	"path/filepath"
	"testing"

	"v2raye/backend-go/internal/domain"
	"v2raye/backend-go/internal/storage"
)

func newImportDedupTestService(t *testing.T) (*Service, *storage.Store) {
	t.Helper()
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	return &Service{store: store}, store
}

func TestImportProfileFromURIUpdatesExistingExactDuplicateProfile(t *testing.T) {
	t.Parallel()

	svc, store := newImportDedupTestService(t)
	existing, err := ParseProfileURI("vless://11111111-1111-1111-1111-111111111111@1.1.1.1:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#node-a", "", "")
	if err != nil {
		t.Fatalf("ParseProfileURI(existing) error = %v", err)
	}
	existing.ID = "existing"
	if err := store.SaveProfiles([]domain.ProfileItem{existing}); err != nil {
		t.Fatalf("SaveProfiles() error = %v", err)
	}

	got, err := svc.ImportProfileFromURI("vless://11111111-1111-1111-1111-111111111111@1.1.1.1:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#node-a")
	if err != nil {
		t.Fatalf("ImportProfileFromURI() error = %v", err)
	}
	if got.ID != "existing" {
		t.Fatalf("ImportProfileFromURI() returned id %q, want existing", got.ID)
	}

	profiles, err := store.LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles() error = %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles count = %d, want 1", len(profiles))
	}
	if profiles[0].Address != "1.1.1.1" {
		t.Fatalf("stored address = %q, want 1.1.1.1", profiles[0].Address)
	}
}

func TestImportProfileFromURIKeepsDistinctProfileWhenAddressDiffers(t *testing.T) {
	t.Parallel()

	svc, store := newImportDedupTestService(t)
	existing, err := ParseProfileURI("vless://11111111-1111-1111-1111-111111111111@1.1.1.1:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#node-a", "", "")
	if err != nil {
		t.Fatalf("ParseProfileURI(existing) error = %v", err)
	}
	existing.ID = "existing"
	if err := store.SaveProfiles([]domain.ProfileItem{existing}); err != nil {
		t.Fatalf("SaveProfiles() error = %v", err)
	}

	got, err := svc.ImportProfileFromURI("vless://11111111-1111-1111-1111-111111111111@2.2.2.2:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#node-a")
	if err != nil {
		t.Fatalf("ImportProfileFromURI() error = %v", err)
	}
	if got.ID == "existing" {
		t.Fatal("ImportProfileFromURI() reused existing ID for different address")
	}

	profiles, err := store.LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles() error = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("profiles count = %d, want 2", len(profiles))
	}
}


func TestDedupeImportedProfilesRemovesOnlyExactDuplicates(t *testing.T) {
	t.Parallel()

	first, err := ParseProfileURI("vless://11111111-1111-1111-1111-111111111111@1.1.1.1:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#node-a", "sub-1", "sub")
	if err != nil {
		t.Fatalf("ParseProfileURI(first) error = %v", err)
	}
	duplicate, err := ParseProfileURI("vless://11111111-1111-1111-1111-111111111111@1.1.1.1:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#node-a", "sub-1", "sub")
	if err != nil {
		t.Fatalf("ParseProfileURI(duplicate) error = %v", err)
	}
	second, err := ParseProfileURI("vless://11111111-1111-1111-1111-111111111111@2.2.2.2:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#node-a", "sub-1", "sub")
	if err != nil {
		t.Fatalf("ParseProfileURI(second) error = %v", err)
	}

	profiles := dedupeImportedProfiles([]domain.ProfileItem{first, duplicate, second})
	if len(profiles) != 2 {
		t.Fatalf("dedupeImportedProfiles() count = %d, want 2", len(profiles))
	}
	if profiles[0].Address != "1.1.1.1" {
		t.Fatalf("dedupeImportedProfiles() kept exact-duplicate address %q, want 1.1.1.1", profiles[0].Address)
	}
	if profiles[1].Address != "2.2.2.2" {
		t.Fatalf("dedupeImportedProfiles() kept second address %q, want 2.2.2.2", profiles[1].Address)
	}
}