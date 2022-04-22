# Blocks

Blocks are a unique aspect of Stencil, blocks enable you to store output between each run of stencil.

## Basic Block

Take the below example:

`README.md.tpl`:

```tpl
# {{ .Config.Name }}

Service {{ .Config.Name }} does a cool thing!

<!--- Block(customDocumentation) --->
{{ file.Block "customDocumentation" }}
<!--- EndBlock(customDocumentation) --->
```

When stencil is ran the first time, it will output:

```md
# my-service

Service my-service does a cool thing!

<!--- Block(customDocumentation) --->
<!--- EndBlock(customDocumentation) --->
```

Pretty basic templating, right. However, if a user adds input within the `<!--- Block(customDocumentation) --->` the next
time stencil is ran, something magical happens:

```md
# my-service

Service my-service does a cool thing!

<!--- Block(customDocumentation) --->

My custom input

<!--- EndBlock(customDocumentation) --->
```

The input is persisted. This is most powerful if we modify the template:

```tpl
# {{ .Config.Name }}

Service {{ .Config.Name }} does a cool thing! It does even cooler things!

<!--- Block(customDocumentation) --->
{{ file.Block "customDocumentation" }}
<!--- EndBlock(customDocumentation) --->

## License

MIT
```

This results in:

```md
# my-service

Service my-service does a cool thing! It does even cooler things!

<!--- Block(customDocumentation) --->

My custom input

<!--- EndBlock(customDocumentation) --->

## License

MIT
```

That input is persisted despite the fact the template was modified around the block.

## Conditional Blocks

You can also take this further with go-templating to allow conditional input:

```tpl
<!--- Block(customDocumentation) --->
{{- if file.Block "customDocumentation" }}
{{ file.Block "customDocumentation" }}
{{- else }}
Fill out this section!
{{- end }}
<!--- EndBlock(customDocumentation) --->
```

The first time this is ran it would output:

```md
<!--- Block(customDocumentation) --->

Fill out this section!

<!--- EndBlock(customDocumentation) --->
```

The user is then able to update the block and change it, keeping it across runs.
