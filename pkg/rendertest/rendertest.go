// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the package documentation and the
// argument fixtures.

// Package rendertest tests the templates of a Stencil module. It renders a
// template without disk, network, git or box access, and returns a Result
// to inspect.
//
// A test package creates one Suite in TestMain. The Suite reads the module
// manifest and resolves the module dependencies once for all tests:
//
//	var suite *rendertest.Suite
//
//	func TestMain(m *testing.M) {
//		suite = rendertest.NewSuite(m)
//		os.Exit(suite.Run())
//	}
//
// Each test renders one template with a TemplateBuilder:
//
//	func TestDeployment(t *testing.T) {
//		t.Parallel()
//
//		result := suite.NewTemplate("deployment.yaml.tpl").
//			WithArgs(rendertest.Arg("name", "my-service")).
//			Run(t)
//
//		if result.Error() != nil {
//			t.Fatal(result.Error())
//		}
//		f := result.File("deployment.yaml")
//		// ...
//	}
//
// Only the templates of the module under test are rendered: the template
// under test and the helper templates passed to TemplateBuilder.WithTemplates.
// The templates of dependency modules are not rendered, so the hook data that
// they would write is set with TemplateBuilder.WithHooks.
//
// Each render uses its own in-memory filesystem, so tests can call
// t.Parallel.
//
// # Snapshots
//
// Result.Snapshot and FileResult.Snapshot compare rendered files with the
// files in testdata/snapshots/<test name>/<file path>. go test only compares
// and never writes snapshots. To write them, run the tests with the
// rendertest_update_snapshots build tag, for example with
// "mise run rendertest:update-snapshots".
//
// # Argument fixtures
//
// ArgOption values set template arguments, and can be combined. A module can
// keep its own fixtures in a package, for example:
//
//	// Package args contains argument fixtures for this module.
//	package args
//
//	func Name(name string) rendertest.ArgOption {
//		return rendertest.Arg("name", name)
//	}
//
//	func Defaults() rendertest.ArgOption {
//		return rendertest.Args(Name("my-service"), rendertest.NestedArg("kubernetes.replicas", 1))
//	}
package rendertest

import "strings"

// ArgOption sets template arguments in the map that it is given.
type ArgOption func(args map[string]any)

// Arg sets the argument key to value.
func Arg(key string, value any) ArgOption {
	return func(args map[string]any) {
		args[key] = value
	}
}

// Args applies opts in order, so a later option overrides an earlier one.
func Args(opts ...ArgOption) ArgOption {
	return func(args map[string]any) {
		for _, opt := range opts {
			opt(args)
		}
	}
}

// NestedArg sets the argument at the dot-separated path to value. It creates
// the maps on the path that do not exist, and replaces values on the path
// that are not maps.
//
//	NestedArg("kubernetes.replicas", 3) // {"kubernetes": {"replicas": 3}}
func NestedArg(path string, value any) ArgOption {
	return func(args map[string]any) {
		keys := strings.Split(path, ".")
		m := args
		for _, k := range keys[:len(keys)-1] {
			next, ok := m[k].(map[string]any)
			if !ok {
				next = map[string]any{}
				m[k] = next
			}
			m = next
		}
		m[keys[len(keys)-1]] = value
	}
}
