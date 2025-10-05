# test/unit

Unit tests now live alongside their owning packages under `cmd/`, `pkg/`, or `internal/` so Go's coverage tooling can attribute statements correctly. This directory remains available for shared fixtures or cross-package helpers if we need them later.
