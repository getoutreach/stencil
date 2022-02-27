# Creating a Template Repository

A template repository is the heart of Stencil. Templates are housed here and, optionally, plugins to interact with stencil.
A template repository consists of a simple `manifest.yaml` ([documentation](https://pkg.go.dev/github.com/getoutreach/stencil@v1.1.1/pkg/configuration#TemplateRepositoryManifest))
that defines the template repository.

## So what is `manifest.yaml`?

```yaml
name: my-first-template-repository
modules: []
arguments:
  my-argument:
    required: true
    type: string
    description: Do a cool thing!
```

The most important, and required, is `name`. This should match the repository name and represents what stencil refers
to your module as. Right below that is `modules`. This is a list of modules, like in `service.yaml` that your repository
depends on. These will be applied (in the virtual filesystem) before your module is, allowing you to overwrite files
that that module created.

Next we have the `arguments` map. This is a simple hash map that allows you to specify arguments that your module has/wants.
`required` denotes that an arguments is required while `type` specifies the type that the argument should be, e.g. `list`, `string`, so forth.
`description` is a user-friendly description of the argument and it's purpose and is required.

That's the core of it! Be sure to checkout the [documentation](https://pkg.go.dev/github.com/getoutreach/stencil@v1.1.1/pkg/configuration#TemplateRepositoryManifest)
for more information.

## Creating the repository

To create a repository simple create a new git repository:

```bash
cd my-first-template-repository
git init
git checkout -b main
git remote add origin <your-git-url>
```

Then create your `manifest.yaml`:

```
name: my-first-template-repository
modules: []
arguments: {}
```

Now push it up: `git push origin main`

Now, in your service just add a block to your `service.yaml`:

```yaml
# Add modules if it doesn't exist, if it does append this
modules:
  - url: <your-git-url>
```

**Note**: For information on the modules spec, see the [documentation](https://pkg.go.dev/github.com/getoutreach/stencil@v1.1.1/pkg/configuration#TemplateRepository)

Now if you run `stencil` you'll use your new repository! :tada:
