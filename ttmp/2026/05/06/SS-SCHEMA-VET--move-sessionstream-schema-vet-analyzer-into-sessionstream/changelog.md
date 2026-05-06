# Changelog

## 2026-05-06

- Initial workspace created

- Added design plan for moving the schema vet analyzer from Pinocchio into Sessionstream and wiring Pinocchio/CoinVault to use `sessionstream-lint`.
- Moved the analyzer into `sessionstream/pkg/analysis/sessionstreamschema`, added `cmd/sessionstream-lint`, wired Sessionstream/Pinocchio/CoinVault `schema-vet` targets, removed the duplicated Pinocchio analyzer, documented usage, and validated the new targets plus targeted downstream tests.
- Noted a follow-up to convert existing Sessionstream tests/systemlab fixtures away from top-level `*structpb.Struct` registrations before broadening Sessionstream self-vet coverage.
