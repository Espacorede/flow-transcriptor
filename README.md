# flow-transcriptor

This is an utility made to transcribe [MediaWiki Flow topic-based talk pages](https://www.mediawiki.org/wiki/Extension:StructuredDiscussions) into old-style pure wikitext pages for the Team Fortress Wiki.

Currently it works by looking for all pages with `flow-board` content models across talk page namespaces and then generating .txt files containing wiki markup styled talk page formats for each one.

The basic part of the code is based on the same old bot that I used to measure "outdatedness" of translations ages ago and it uses [jsonparser](https://github.com/buger/jsonparser) for easy reading of API JSON responses. (As well as [sanitize](https://github.com/kennygrant/sanitize) for saving filenames)

