---
title: service.yaml
linktitle: Service Manifest
description: The service.yaml passes arguments and defines which modules an application should use
date: 2022-05-04
publishdate:  2022-05-04
menu:
  docs:
    parent: "reference"
    weight: 4
categories: [application]
keywords: [application, service manifest]
toc: true
---

## What is a `service.yaml`?

A `service.yaml` can be thought as the specification for an application based on the modules being used. It defines the modules an application uses and the arguments to pass to them. 

## What are the fields in a `service.yaml`

* `name`: The name of the application
* `arguments`: The arguments to pass to the modules. This is a map of key value pairs.
* `modules`: The modules to use. This is a list of objects containing a `name` and a, optionally, `version` field to use of this module.
* `replacements`: A key/value of importPath to replace with another source. This is useful for replacing modules with a different version or local testing. Source should be a valid URL, import path, or file path on disk.