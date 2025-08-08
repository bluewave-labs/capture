package test

import (
	"testing"

	"github.com/mstrYoda/go-arctest/pkg/arctest"
)

func TestArchitecture(t *testing.T) {
	arch, err := arctest.New("../")
	if err != nil {
		t.Fatalf("Failed to create architecture: %v", err)
	}

	err = arch.ParsePackages()
	if err != nil {
		t.Fatalf("Failed to parse packages: %v", err)
	}

	// Architecture Rule: CMD should not depend on internal/handler
	cmdDoesNotDependOnHandler, err := arch.DoesNotDependOn("cmd.*$", "internal/handler.*$")
	if err != nil {
		t.Fatalf("Failed to create dependency rule: %v", err)
	}

	// Validate dependencies
	valid, violations := arch.ValidateDependenciesWithRules([]*arctest.DependencyRule{
		cmdDoesNotDependOnHandler,
	})
	if !valid {
		for _, violation := range violations {
			t.Errorf("Dependency violation: %s", violation)
		}
	}
}
