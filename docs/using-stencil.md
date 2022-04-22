# Getting Started with Stencil

Stencil, for an end user is pretty simple. To start using stencil you'll need to create a `service.yaml`:

```yaml
name: name-of-my-thing
arguments: {}
modules: []
```

A basic service.yaml, like the above, will successfully run, but it won't do anything particularly exciting as it has no templates.

From here you could [create your own template repository](./creating-a-template-repository.md) or you could use a module provided by someone else.

Assuming you already have a valid template repository, either self-created or from a remote repository, you can easily
start using it by adding it to `modules`:

```yaml
modules:
  - name: github.com/getoutreach/stencil-base
```

Next time you run stencil, you'll get the templates from that repository!

## Using Local Templates

If you ever want to replace remote templates with a testing version, or just test before you release a template you can
easily do that by adding the module to the `replacements` key in the `service.yaml`:

```yaml
modules:
  - name: github.com/getoutreach/stencil-base
replacements:
  github.com/getoutreach/stencil-base: ../my-local-copy/stencil-base
```

That's it! You'll be using your local copy instead :smile:
