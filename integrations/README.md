# Integrations

Integration directories are organized by tool or use case. Each may contain:

- domain vocabulary and contract packs;
- evidence-adapter specifications;
- synthetic fixtures and validation commands;
- adoption criteria and open decisions;
- language-specific examples subordinate to the integration design.

An integration directory does not imply that an MCP server or adapter is
implemented. Current substantive integration:

- [Puppet](puppet/README.md) — shared Hiera vocabulary, Puppet-aware evidence,
  catalog relationships, and staged change validation.

Future integrations should be added only with reviewed, useful content rather
than empty placeholder directories.

Cross-integration architecture and the consuming-repository manifest
layout live in [DESIGN.md](../DESIGN.md).
