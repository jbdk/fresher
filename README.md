**in development**

# Fresher

This is a rewrite of `github.com/pilu/fresh` (also known as `github.com/gravityblast/fresh`) to add better configuration, better code documenation, and hopefully improve performance a bit.


## Motivation:
`fresh` works well, but:
  - The configuration file is in a self-defined format, it does not use YAML, JSON, or something easily understood and commonly understood. 
  - The configuration options also lack the ability to support build tags or other such features.
  - The code base is nearly totally undocumented so adding documentation should help future maintance. 
  - Does not use go modules.
  - Performance is quite good, but hopefully we can make it a bit faster.

## FAQs: 

### Why not just use `air` (https://github.com/cosmtrek/air)?
Simply put, in my testing and usage `fresh` is much faster than `air`. This is odd since `air` is a fork of `fresh` more recent maintenance. However, it seems `air` is much "heavier" and there is more of a focus on tooling (testing, dev. tools, etc.) rather than the underlying functionality.

