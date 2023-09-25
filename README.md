# check-translations

## Usage

```
$ go run . ./folder/with/translations/
```

## How does it work?

The program scans the given folder for JSON files, reads them, runs several checks on them and gives a report in case of any issues. The files are expected to be at the top of the folder itself, not nested in other folders.
```
$ find localizations/
localizations/en.json
localizations/cs.json
localizations/no.json
localizations/ru.json
...
```

Every JSON file is expected to have the following structure
```
{
    "translation.key.one": "Do something",
    "translation.key.two": "Do something else",
    "translation.key.three": "Do something different with some <b>HTML tags</b>",
    "translation.key.four": "Do something with a $variable$",
    ...
}
```
where the keys are used as translation identifiers, and the values are the actual texts. While the identifiers stay the same in all the files, the values are translated. The values can also include variables, formatted as `$variable$`, which themselves must **not** be translated. Additionally, the values can include HTML tags.

The `en.json` file is read first and used as the reference for all the other languages. When the program is run, it uses the identifiers from `en.json` to go through all the translations in all the files that match the `??.json` glob and performs the following checks:

* Go through all the variables in the reference english text and check whether they are all present in the translated text, and none of them has been changed (likely translated).
* Go through all the texts and check whether all HTML tags are properly closed. Texts with no tags in them are considered valid HTML.

## GitHub Actions

The checks are meant to be used from CI. The Go toolchain is easy and fast to set up and the program itself compiles and runs reasonably quickly.

To run the checks from a GitHub Actions workflow, use the following snippet:
```
...
jobs:
  check-translations:
    steps:
    ...
      - name: Set up Go
        uses: actions/setup-go
        with:
          go-version: '1.21'

      - name: Translation tests
        run: |
          go install github.com/scrive/check-translations@latest
          $HOME/go/bin/check-translations ./localizations/
    ...
...
```
