# Example Verification

Verified on Windows with:

- OS: Windows amd64
- Go: `go1.26.4 windows/amd64`
- Node: `v24.14.0`

## Go Library

The Go library builds and tests successfully:

```bash
go test .
go test ./...
```

The root pure-Go tests require shared test helpers for encryption/decryption,
boolean gates, bitwise integer addition, bigint round trips, key/ciphertext
serialization, and encrypted RNG checks. Those helpers are supplied by
`test_helpers_test.go`.

## Examples

The example directories under `examples/` are Hardhat/TypeScript projects, not
Go packages. Each example has a `package.json` with `compile` and `test`
scripts that call Hardhat:

| Example | Package | Expected verification command |
| --- | --- | --- |
| `examples/content-provenance` | `@luxfhe/example-content-provenance` | `npm install && npm test` |
| `examples/data-seal` | `@luxfhe/example-data-seal` | `npm install && npm test` |
| `examples/encrypted-crdt` | `@luxfhe/example-encrypted-crdt` | `npm install && npm test` |
| `examples/shadow-governance` | `@luxfhe/example-shadow-governance` | `npm install && npm test` |

No example directory currently includes a lockfile (`package-lock.json`,
`pnpm-lock.yaml`, or `yarn.lock`), and this verification environment does not
provide `npm`, `npx`, `pnpm`, `yarn`, or `corepack`. Because of that, the
Hardhat examples could not be installed or executed here. A follow-up pass
should run the commands above in an environment with a Node package manager.
