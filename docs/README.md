[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](https://getoutreach.github.io/stencil/contribute/documentation/)

# Stencil Docs

Documentation site for [Stencil](https://github.com/getoutreach/stencil), a programmable templating engine for Microservices and more.

## Contributing

We welcome contributions to Stencil of any kind including documentation, suggestions, bug reports, pull requests etc. Also check out our [contribution guide](https://geoutreach.github.io/stencil/contribute/documentation/). We would love to hear from you. 

Note that this repository contains solely the documentation for Stencil. For contributions that aren't documentation-related please refer to the [Stencil](https://github.com/getoutreach/stencil) repository. 

Spelling fixes are most welcomed, and if you want to contribute longer sections to the documentation, it would be great if you had the following criteria in mind when writing:

* Short is good. People go to the library to read novels. If there is more than one way to _do a thing_ in Stencil, describe the current _best practice_ (avoid "… but you can also do …" and "… in older versions of Stencil you had to …".
* For example, try to find short snippets that teaches people about the concept. If the example is also useful as-is (copy and paste), then great. Don't list long and similar examples just so people can use them on their sites.
* We want to be friendly towards users across that world, so easy to understand and [simple English](https://simple.wikipedia.org/wiki/Basic_English) is good.

## Branches

* The `main` branch is where the site is automatically built from, and is the place to put changes relevant to the current Hugo version.

## Build

To view the documentation site locally, you need to clone this repository:

```bash
git clone https://github.com/getoutreach/stencil-docs.git
```

Then to view the docs in your browser, run Hugo and open up the link:

```bash
▶ hugo server

Started building sites ...
.
.
Serving pages from memory
Web Server is available at http://localhost:1313/ (bind address 127.0.0.1)
Press Ctrl+C to stop
```


## Credit

This is _heavily_ based on/forked from the hugo docs site: https://github.com/gohugoio/hugoDocs. Thanks to the authors of hugo for making this possible and providing a great framework!