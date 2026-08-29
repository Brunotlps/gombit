# Domain validation, transactions, and optimistic locking

Gombit gives domain rules a single home that both the API and the admin write
paths run, so an invariant enforced once cannot be bypassed by the other
surface (issue #224).

## Model validation (`database.Validator`)

Implement `Validate(ctx, tx)` on a model to enforce rules that cross fields,
rows, or models. The framework runs it via a GORM callback on **every**
create/update — the generated resource handler (`db.Create` / `db.Save`) and the
admin data plane both go through it, inside the write transaction:

```go
func (r Rental) Validate(ctx context.Context, tx *gorm.DB) error {
	if r.EndsAt.Before(r.StartsAt) {
		return database.NewValidationError(
			"The request contains invalid fields.",
			map[string][]string{"ends_at": {"must be after starts_at"}},
		)
	}
	// tx runs the current transaction — query it for cross-row invariants:
	var overlaps int64
	if err := tx.Model(&Rental{}).
		Where("engine_id = ? AND id <> ? AND starts_at < ? AND ends_at > ?",
			r.EngineID, r.ID, r.EndsAt, r.StartsAt).
		Count(&overlaps).Error; err != nil {
		return err
	}
	if overlaps > 0 {
		return database.NewValidationError("Ownership windows must not overlap.", nil)
	}
	return nil
}
```

A returned `*database.ValidationError` maps to a D10 **422** `validation_error`
(with `fields`) through `database.MapPersistError`, which the generated handlers
and the admin already call. Any other error aborts the write (and rolls back the
surrounding transaction) as a 500.

`Validate` is defined structurally: a model implements it just by declaring the
method — no framework import is required (the signature uses `context.Context`
and `*gorm.DB`, which models already have).

## Transactions (`app.Tx`)

`app.Tx(ctx, fn)` runs `fn` in one transaction, committing on `nil` and rolling
back on any error or panic. It is the home for transactional multi-model writes:

```go
err := app.Tx(ctx, func(tx *gorm.DB) error {
	if err := tx.Create(&order).Error; err != nil {
		return err
	}
	return tx.Model(&inventory).Update("count", gorm.Expr("count - ?", n)).Error
})
```

`Validate` hooks fire inside this transaction too, so a rule checked in
`Validate` cannot commit next to a change that violates it.

## Optimistic locking

Give a model an integer `version` column and the admin update path enforces
optimistic locking automatically: the update matches on the version read (or a
`version` supplied in the request body, a stricter precondition based on what
the client last saw), bumps it, and returns **409 Conflict** when no row
matches — instead of a silent last-write-wins:

```go
type Account struct {
	gorm.Model
	Balance int
	Version int `gorm:"default:1"`
}
```

Two concurrent PATCHes of the same row can no longer both win: the loser matches
zero rows and gets a 409. Reload and retry. For resource handlers, apply the
same guard with GORM inside `app.Tx`:

```go
res := tx.Model(&acct).Where("version = ?", expected).
	Updates(map[string]any{"balance": newBalance, "version": expected + 1})
if res.RowsAffected == 0 {
	return contract.WithContext(ctx, contract.Conflict("modified by another request"))
}
```
