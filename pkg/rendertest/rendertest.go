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
//
// # Migrating from stenciltest
//
// Packages stenciltest and rendertest can be used in the same Go module, so a
// module can migrate one test at a time. To migrate a module:
//
//  1. Add a TestMain that creates the Suite, as shown at the top of this page.
//  2. Convert each test, as described below.
//  3. Optionally, move the arguments that tests share into a fixture package.
//  4. Write the snapshots with the command in the Snapshots section, then
//     review and commit the files in testdata/snapshots.
//  5. When all tests of the module are converted, remove the imports of
//     stenciltest, the .snapshots directory and the cupaloy dependency.
//
// A stenciltest test:
//
//	func TestDeployment(t *testing.T) {
//		st := stenciltest.New(t, "deployment.yaml.tpl")
//		st.Args(map[string]any{"name": "my-service"})
//		st.Run(stenciltest.RegenerateSnapshots())
//	}
//
// becomes:
//
//	func TestDeployment(t *testing.T) {
//		result := suite.NewTemplate("deployment.yaml.tpl").
//			WithRawArgs(map[string]any{"name": "my-service"}).
//			Run(t)
//
//		result.Snapshot(t)
//	}
//
// The common calls of a stenciltest.Template map to these:
//
//   - stenciltest.New(t, path, helpers...) is suite.NewTemplate(path), with
//     TemplateBuilder.WithTemplates(helpers...) for the helper templates. To
//     load all templates of the module as helpers, pass FindTemplates(t, ".")
//     to WithTemplates.
//   - Template.Args(args) is TemplateBuilder.WithRawArgs(args), or
//     TemplateBuilder.WithArgs with argument fixtures.
//   - Template.Ext(name, ext) is TemplateBuilder.WithExtension(name, ext).
//   - Template.ErrorContains(msg) is a check of Result.Error, which is the
//     render error.
//   - Template.Run(stenciltest.RegenerateSnapshots()) is TemplateBuilder.Run(t)
//     followed by Result.Snapshot(t).
//
// Three behaviors differ:
//
//   - stenciltest overwrites a changed snapshot on every run where the CI
//     environment variable is empty. That run fails, and the next run passes
//     without a review of the change. rendertest writes snapshots only when it
//     is built with the rendertest_update_snapshots build tag.
//   - stenciltest renders the templates of every module in the dependency
//     tree, so those templates write the hook values of dependency modules.
//     rendertest renders only the template under test and the helper
//     templates, and TemplateBuilder.WithHooks sets those hook values.
//   - Snapshots are in testdata/snapshots/<test name>/<file path>, with the
//     exact contents of the file, and not in .snapshots.
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
