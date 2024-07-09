package interfaces

import "testing"

func TestLoadPackage(t *testing.T) {
	// Load the package.
	pkg, err := loadSsz()
	if err != nil {
		t.Fatal(err)
	}

	// Check the package.
	if pkg == nil {
		t.Fatal("expected a package")
	}
}
