---
title: Contribute to the Stencil Docs
linktitle: Documentation
description: Documentation is an integral part of any open source project. The Stencil docs are as much a work in progress as the source it attempts to cover.
date: 2022-05-02
publishdate: 2022-05-02
lastmod: 2022-05-02
categories: [contribute]
keywords: [docs, documentation, community, contribute]
menu:
  docs:
    parent: "contribute"
    weight: 20
weight: 20
sections_weight: 20
draft: false
aliases: [/contribute/docs/]
toc: true
---

## Create Your Fork

It's best to make changes to the Stencil docs on your local machine to check for consistent visual styling. Make sure you've created a fork of [stencil](https://github.com/getoutreach/stencil) on GitHub and cloned the repository locally on your machine. For more information, you can see [GitHub's documentation on "forking"][ghforking] or follow along with [Stencil's development contribution guide][stencildev].

You can then create a separate branch for your additions. Be sure to choose a descriptive branch name that best fits the type of content. The following is an example of a branch name you might use for adding a new function:

```
git checkout -b jon-doe-function-addition
```

## Add New Content

The Stencil docs make heavy use of Hugo's [archetypes][] feature. All content sections in Stencil documentation have an assigned archetype.

Adding new content to the Stencil docs follows the same pattern, regardless of the content section:

```
hugo new <DOCS-SECTION>/<new-content-lowercase>.md
```

### Add a New Function

Once you have cloned the Stencil repository, you can create a new function via the following command. Keep the file name lowercase.

```
hugo new functions/newfunction.md
```

The archetype for `functions` according to the Stencil docs is as follows:

{{< code file="archetypes/functions.md" >}}
{{< readfile file="/archetypes/functions.md">}}
{{< /code >}}

#### New Function Required Fields

Here is a review of the front matter fields automatically generated for you using `hugo new functions/*`:

**_`title`_**
: this will be auto-populated in all lowercase when you use `hugo new` generator.

**_`linktitle`_**
: the function's actual casing (e.g., `replaceRE` rather than `replacere`).

**_`description`_**
: a brief description used to populate the [Functions Quick Reference](/functions/).

`categories`
: currently auto-populated with 'functions` for future-proofing and portability reasons only; ignore this field.

`tags`
: only if you think it will help end users find other related functions

`signature`
: this is a signature/syntax definition for calling the function (e.g., `apply SEQUENCE FUNCTION [PARAM...]`).

`workson`
: acceptable values include `lists`,`taxonomies`, `terms`, `groups`, and `files`.

`stencilversion`
: the version of Stencil that will ship with this new function.

`relatedfuncs`
: other [templating functions][] you feel are related to your new function to help fellow Stencil users.

`{{.Content}}`
: an extended description of the new function; examples are not only welcomed but encouraged.

In the body of your function, expand the short description used in the front matter. Include as many examples as possible, and leverage the Stencil docs [`code` shortcode](#add-code-blocks). If you are unable to add examples but would like to solicit help from the Stencil community, add `needsexample: true` to your front matter.

## Add Code Blocks

Code blocks are crucial for providing examples of Stencil's new features to end users of the Stencil docs. Whenever possible, create examples that you think Stencil users will be able to implement in their own projects.

### Standard Syntax

Across many pages on the Stencil docs, the typical triple-back-tick markdown syntax (` ``` `) is used. If you do not want to take the extra time to implement the following code block shortcodes, please use standard GitHub-flavored markdown. The Stencil docs use a version of [highlight.js](https://highlightjs.org/) with a specific set of languages.

Your options for languages are `xml`/`html`, `go`/`golang`, `md`/`markdown`/`mkd`, `handlebars`, `apache`, `toml`, `yaml`, `json`, `css`, `asciidoc`, `ruby`, `powershell`/`ps`, `scss`, `sh`/`zsh`/`bash`/`git`, `http`/`https`, and `javascript`/`js`.

````
```
<h1>Hello world!</h1>
```
````

### Code Block Shortcode

The Stencil documentation comes with a very robust shortcode for adding interactive code blocks.

{{% note %}}
With the `code` shortcodes, _you must include triple back ticks and a language declaration_. This was done by design so that the shortcode wrappers were easily added to legacy documentation and will be that much easier to remove if needed in future versions of the Stencil docs.
{{% /note %}}

### `code`

`code` is the Stencil docs shortcode you'll use most often. `code` requires has only one named parameter: `file`. Here is the pattern:

```
{{%/* code file="smart/file/name/with/path.html" download="download.html" copy="true" */%}}
A whole bunch of coding going on up in here!
{{%/* /code */%}}
```

The following are the arguments passed into `code`:

**_`file`_**
: the only _required_ argument. `file` is needed for styling but also plays an important role in helping users create a mental model around Stencil's directory structure. Visually, this will be displayed as text in the top left of the code block.

`download`
: if omitted, this will have no effect on the rendered shortcode. When a value is added to `download`, it's used as the filename for a downloadable version of the code block.

`copy`
: a copy button is added automatically to all `code` shortcodes. If you want to keep the filename and styling of `code` but don't want to encourage readers to copy the code (e.g., a "Do not do" snippet in a tutorial), use `copy="false"`.

#### Example `code` Input

This example HTML code block tells Stencil users the following:

1. This file _could_ live in `layouts/_default`, as demonstrated by `layouts/_default/single.html` as the value for `file`.
2. This snippet is complete enough to be downloaded and implemented in a Stencil project, as demonstrated by `download="single.html"`.

```
{{</* code file="layouts/_default/single.html" download="single.html" */>}}
{{ define "main" }}
<main>
    <article>
        <header>
            <h1>{{.Title}}</h1>
            {{with .Params.subtitle}}
            <span>{{.}}</span>
        </header>
        <div>
            {{.Content}}
        </div>
        <aside>
            {{.TableOfContents}}
        </aside>
    </article>
</main>
{{ end }}
{{</* /code */>}}
```

##### Example 'code' Display

The output of this example will render to the Stencil docs as follows:

{{< code file="layouts/_default/single.html" download="single.html" >}}
{{ define "main" }}

<main>
    <article>
        <header>
            <h1>{{.Title}}</h1>
            {{with .Params.subtitle}}
            <span>{{.}}</span>
        </header>
        <div>
            {{.Content}}
        </div>
        <aside>
            {{.TableOfContents}}
        </aside>
    </article>
</main>
{{ end }}
{{< /code >}}

<!-- #### Output Code Block

The `output` shortcode is almost identical to the `code` shortcode but only takes and requires `file`. The purpose of `output` is to show *rendered* HTML and therefore almost always follows another basic code block *or* and instance of the `code` shortcode:

```
{{%/* output file="posts/my-first-post/index.html" */%}}
<h1>This is my First Stencil Blog Post</h1>
<p>I am excited to be using Stencil.</p>
{{%/* /output */%}}
```

The preceding `output` example will render as follows to the Stencil docs:

{{< output file="posts/my-first-post/index.html" >}}
<h1>This is my First Stencil Blog Post</h1>
<p>I am excited to be using Stencil.</p>
{{< /output >}} -->

## Blockquotes

Blockquotes can be added to the Stencil documentation using [typical Markdown blockquote syntax][bqsyntax]:

```
> Without the threat of punishment, there is no joy in flight.
```

The preceding blockquote will render as follows in the Stencil docs:

> Without the threat of punishment, there is no joy in flight.

However, you can add a quick and easy `<cite>` element (added on the client via JavaScript) by separating your main blockquote and the citation with a hyphen with a single space on each side:

```
> Without the threat of punishment, there is no joy in flight. - [Kobo Abe](https://en.wikipedia.org/wiki/Kobo_Abe)
```

Which will render as follows in the Stencil docs:

> Without the threat of punishment, there is no joy in flight. - [Kobo Abe][abe]

## Admonitions

**Admonitions** are common in technical documentation. The most popular is that seen in [reStructuredText Directives][sourceforge]. From the SourceForge documentation:

> Admonitions are specially marked "topics" that can appear anywhere an ordinary body element can. They contain arbitrary body elements. Typically, an admonition is rendered as an offset block in a document, sometimes outlined or shaded, with a title matching the admonition type. - [SourceForge][sourceforge]

The Stencil docs contain three admonitions: `note`, `tip`, and `warning`.

### `note` Admonition

Use the `note` shortcode when you want to draw attention to information subtly. `note` is intended to be less of an interruption in content than is `warning`.

#### Example `note` Input

{{< code file="note-with-heading.md" >}}
{{%/* note */%}}
Here is a piece of information I would like to draw your **attention** to.
{{%/* /note */%}}
{{< /code >}}

#### Example `note` Output

{{< output file="note-with-heading.html" >}}
{{% note %}}
Here is a piece of information I would like to draw your **attention** to.
{{% /note %}}
{{< /output >}}

#### Example `note` Display

{{% note %}}
Here is a piece of information I would like to draw your **attention** to.
{{% /note %}}

### `tip` Admonition

Use the `tip` shortcode when you want to give the reader advice. `tip`, like `note`, is intended to be less of an interruption in content than is `warning`.

#### Example `tip` Input

{{< code file="using-tip.md" >}}
{{%/* tip */%}}
Here's a bit of advice to improve your productivity with Stencil.
{{%/* /tip */%}}
{{< /code >}}

#### Example `tip` Output

{{< output file="tip-output.html" >}}
{{% tip %}}
Here's a bit of advice to improve your productivity with Stencil.
{{% /tip %}}
{{< /output >}}

#### Example `tip` Display

{{% tip %}}
Here's a bit of advice to improve your productivity with Stencil.
{{% /tip %}}

### `warning` Admonition

Use the `warning` shortcode when you want to draw the user's attention to something important. A good usage example is for articulating breaking changes in Stencil versions, known bugs, or templating "gotchas."

#### Example `warning` Input

{{< code file="warning-admonition-input.md" >}}
{{%/* warning */%}}
This is a warning, which should be reserved for _important_ information like breaking changes.
{{%/* /warning */%}}
{{< /code >}}

#### Example `warning` Output

{{< output file="warning-admonition-output.html" >}}
{{% warning %}}
This is a warning, which should be reserved for _important_ information like breaking changes.
{{% /warning %}}
{{< /output >}}

#### Example `warning` Display

{{% warning %}}
This is a warning, which should be reserved for _important_ information like breaking changes.
{{% /warning %}}

{{% note "Pull Requests and Branches" %}}
Similar to [contributing to Stencil development](/contribute/development/), the Stencil team expects you to create a separate branch/fork when you make your contributions to the Stencil docs.
{{% /note %}}

[abe]: https://en.wikipedia.org/wiki/Kobo_Abe
[archetypes]: /content-management/archetypes/
[bqsyntax]: https://github.com/adam-p/markdown-here/wiki/Markdown-Cheatsheet#blockquotes
[charcount]: https://www.lettercount.com/
[ghforking]: https://help.github.com/articles/fork-a-repo/
[stencildev]: /contribute/development/
[sourceforge]: https://docutils.sourceforge.io/docs/ref/rst/directives.html#admonitions
[templating function]: /functions/
