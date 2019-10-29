# third_party/

This directory serves the same purpose as a Go vendor/ except that the third-party source code is often just a small part of an existing open-source Go package, or even just a snippet from an origin like Stack Overflow or Github Gist.

The intent is to make a best effort at maintaining a delineation between first- and third-party source code that is clearly communicated both in the file structure (separate "third_party" file tree) and import paths.

# Approach

- If the origin provided licensing details about the third-party code, those details are retained/replicated (e.g. license in a file heading or separate LICENSE file).
  - If the origin did not provide them, at least the origin URL and available author details are provided.
- If modifications to the third-party code were made, a best effort is made to note that fact and summarize them.
- Unless otherwise noted, the modifications to the third-party code are MIT licensed.

# Report

If any licensing information is incorrect, missing, or unclear, please file an issue so I can make corrections.
