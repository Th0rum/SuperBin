
## Development

Requirements:

- go
- [sqlc](https://github.com/sqlc-dev/sqlc) for database code generation
- Make (GNU)

### Commands

- `make build` - build project (dev mode)
- `make run` - run local dev version
- `make generate` - trigger code generation 



### Notes

- Run `make generate` after you added new migration or query to update bindings.