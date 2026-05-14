# StrataFS Documentation

The source for the StrataFS documentation site, built with [MkDocs](https://www.mkdocs.org/) and the [Material](https://squidfunk.github.io/mkdocs-material/) theme.

## Local preview

```bash
cd documentation
pip install -r requirements.txt
mkdocs serve
```

Open <http://127.0.0.1:8000>.

## Strict build

```bash
mkdocs build --strict
```

`--strict` fails the build on broken internal links — run it before opening a docs PR.

## Publishing

Pushes to `main` that touch `documentation/**` trigger `.github/workflows/docs.yml`, which builds the site and deploys it to GitHub Pages.
