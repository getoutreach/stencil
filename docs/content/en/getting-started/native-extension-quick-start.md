---
title: Native Extension Quick Start
linktitle: Native Extension Quick Start
description: Create a native extension and use it in an application
date: 2022-05-02
publishdate: 2022-05-02
categories: [getting started]
keywords: [quick start, usage]
authors: [jaredallard]
menu:
  docs:
    parent: "getting-started"
    weight: 30
toc: true
---

{{% note %}}
This quick start assumes you're familiar with stencil and module usage already. If you aren't be sure to go through the [reference documentation](/stencil/reference/) or the other quick starts here before proceeding. You've been warned!
{{% /note %}}

# What is a Native Extension?

Native extensions are special module types that don't use go-templates to integrate with stencil. Instead they expose template functions written in another language that can be called by stencil templates.

# How to create a Native Extension

This quick start will focus on creating a Go native extension. While other languages may work as well, there currently is no official documentation or support for those languages (if you're interested in another language please contribute it back!).

## Step 1: Create a Native Extension

Much like a module we're going to use the [`stencil create module`] command to create a native extension.

{{< code file="create-module.sh" >}}
mkdir helloworld; cd helloworld
stencil create module --native-extension github.com/yourorg/helloworld
{{< /code >}}

However, instead of using the `templates/` directory we're going to create a `plugin/` directory.

{{< code file="setup-dirs.sh" >}}
rm templates; mkdir plugin
{{< /code >}}

Now that we've created the `plugin/` directory we're going to created a simple `plugin.go` file that'll implement the `Implementation` interface and prints `helloWorld` when the `helloWorld` function is called.

{{< code file="plugin.go" >}}
package main

import (
"fmt"

    "github.com/getoutreach/stencil/pkg/extensions/apiv1"
    "github.com/sirupsen/logrus"

)

// _ is a compile time assertion to ensure we implement
// the Implementation interface
var _ apiv1.Implementation = &TestPlugin{}

type TestPlugin struct{}

func (tp *TestPlugin) GetConfig() (*apiv1.Config, error) {
return &apiv1.Config{}, nil
}

func (tp *TestPlugin) ExecuteTemplateFunction(t *apiv1.TemplateFunctionExec) (interface{}, error) {
if t.Name == "helloWorld" {
return "helloWorld"
}

    return nil, nil

}

func (tp *TestPlugin) GetTemplateFunctions() ([]*apiv1.TemplateFunction, error) {
return []\*apiv1.TemplateFunction{
{
Name: "helloWorld",
},
}, nil
}

func helloWorld() (interface{}, error) {
fmt.Println("👋 from the test plugin")
return "hello from a plugin!", nil
}

func main() {
err := apiv1.NewExtensionImplementation(&TestPlugin{})
if err != nil {
logrus.WithError(err).Fatal("failed to start extension")
}
}
{{< /code >}}

Now lets run `make` to create the binary at `bin/plugin` so we can consume it in a test application.

## Step 2: Using in a Test Module

Let's create a `testmodule` to consume the native extension.

{{< code file="create-test-module.sh" copy=true >}}
mkdir testmodule; cd testmodule
stencil create module github.com/yourorg/testmodule
{{< /code >}}

Now let's create a `hello.txt.tpl` that consumes the `helloWorld` function.

{{< code file="hello.txt.tpl" copy=true >}}
{{ extensions.Call "github.com/yourorg/helloworld" "helloWorld" }}
{{< /code >}}

Ensure that the `manifest.yaml` for this module consumes the native extension:

{{< code file="manifest.yaml" copy=true >}}
name: testmodule
modules:

- name: github.com/yourorg/helloworld
  {{< /code >}}

## Step 3: Running the Test Module

Now, in order to test the native extension and the module consuming it we'll need to create a test application.

{{< code file="create-test-module.sh" copy=true >}}
mkdir testapp; cd testapp
cat > service.yaml <<EOF
name: testapp
modules:

- name: github.com/yourorg/testmodule
  replacements: # Note: Replace these directories with their actual paths. This assumes their # right behind our application in the directory tree.
  github.com/yourorg/helloworld: ../helloworld
  github.com/yourorg/testmodule: ../testmodule
  {{< /code >}}

Now, if we run `stencil` we should get a `hello.txt` file in our test application.

```bash
testapp ❯ stencil
INFO[0000] stencil v1.14.2
INFO[0000] Fetching dependencies
INFO[0002]  -> github.com/yourorg/helloworld local
INFO[0002] Loading native extensions
INFO[0002] Rendering templates
INFO[0002] Writing template(s) to disk
INFO[0002]   -> Created hello.txt

testapp ❯ cat hello.txt
helloWorld
```

Success! :tada:

# Reflection

To reflect, we've created a `hello.txt.tpl` file that calls the `helloWorld` function in the native extension we implemented.

Releasing is all handled by the templates in `stencil-base` and `stencil-template-base` that were created when you ran `stencil create`
