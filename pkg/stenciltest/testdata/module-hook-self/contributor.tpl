{{- file.Skip "test fixture: contributes to a module hook owned by the module under test" }}
{{ stencil.AddToModuleHook "testing" "selfHook" (list "one-value") }}
