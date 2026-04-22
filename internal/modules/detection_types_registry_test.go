package modules_test

// This test lives in an external _test package so it can blank-import
// the concrete module packages without creating an import cycle from
// the modules package itself. Its sole purpose is to fail the build
// when KnownDetectionTypes drifts from what the registry actually has
// after every module's init() has run.

import (
	"sort"
	"testing"

	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"

	_ "github.com/SiriusScan/app-agent/internal/modules/filecontent"
	_ "github.com/SiriusScan/app-agent/internal/modules/filehash"
	_ "github.com/SiriusScan/app-agent/internal/modules/filesearch"
	_ "github.com/SiriusScan/app-agent/internal/modules/versioncmd"
)

func TestKnownDetectionTypesMatchesRegistry(t *testing.T) {
	known := append([]string(nil), modules.KnownDetectionTypeNames()...)
	registered := registry.List()
	sort.Strings(registered)

	knownSet := toSet(known)
	regSet := toSet(registered)

	var missingFromKnown, missingFromRegistry []string
	for _, name := range registered {
		if _, ok := knownSet[name]; !ok {
			missingFromKnown = append(missingFromKnown, name)
		}
	}
	for _, name := range known {
		if _, ok := regSet[name]; !ok {
			missingFromRegistry = append(missingFromRegistry, name)
		}
	}

	if len(missingFromKnown) > 0 {
		t.Errorf(
			"modules registered but missing from KnownDetectionTypes (add them to internal/modules/detection_types.go): %v",
			missingFromKnown,
		)
	}
	if len(missingFromRegistry) > 0 {
		t.Errorf(
			"types in KnownDetectionTypes have no registered module (either remove them or blank-import the module package in this test): %v",
			missingFromRegistry,
		)
	}

	if t.Failed() {
		t.Logf("registered modules: %v", registered)
		t.Logf("known detection types: %v", known)
	}
}

func toSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}
