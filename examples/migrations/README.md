# Migration Example

Run the example command from the repository root:

```sh
gombit db makemigrations create_products \
  --driver sqlite \
  --model github.com/LAA-Software-Engineering/gombit/examples/migrations/internal/product.Product
```

This demonstrates the feature-package model path used by the temporary Atlas
Program Mode loader. The generated migration files are written to
`database/migrations` unless `--dir` is provided.
