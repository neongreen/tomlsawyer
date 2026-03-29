# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - Unreleased

### Added

- Comment-preserving TOML parsing and serialization
- Get/Set/Delete/Has API
- Keys() method for discovering table children (fixes neongreen/mono#337)
- Keys() works with dotted key prefixes
- Has() works for sections and dotted key prefixes
- TopLevelKeys() method
- Move() method (rename/move sections and keys, like Unix mv)
- ApplyMap/ReplaceMap for bulk operations
- WriteFile helper
- Broad TOML v1.0.0 support
- Golden tests with go-cmp diffs
- Key expressibility tests
- Testable Go doc examples
