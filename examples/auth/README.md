# Bearer auth example

Minimal `framework.App` with JWT secret + SQLite. `framework.New` mounts
`POST /api/v1/auth/{register,login,refresh,logout}` and `GET /api/v1/me`.

## Run

```sh
go run ./examples/auth
```

Interactive docs: [http://127.0.0.1:8080/docs](http://127.0.0.1:8080/docs).

## E2E (D10 envelopes)

```sh
# register
curl -sS -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct-horse"}'

# login
LOGIN=$(curl -sS -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct-horse"}')
echo "$LOGIN"

ACCESS=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["access_token"])' <<<"$LOGIN")
REFRESH=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["refresh_token"])' <<<"$LOGIN")

# protected route
curl -sS http://127.0.0.1:8080/api/v1/me -H "Authorization: Bearer $ACCESS"

# rotate refresh (old refresh is then invalid)
ROTATED=$(curl -sS -X POST http://127.0.0.1:8080/api/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH\"}")
echo "$ROTATED"
ACCESS=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["access_token"])' <<<"$ROTATED")
REFRESH=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["refresh_token"])' <<<"$ROTATED")

# logout (revokes the current refresh token and bound access JWT)
curl -sS -X POST http://127.0.0.1:8080/api/v1/auth/logout \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH\"}"
```

The development JWT secret in `main.go` is not a production secret.
Production config rejects secrets shorter than 32 characters.

See [`docs/auth.md`](../../docs/auth.md).
