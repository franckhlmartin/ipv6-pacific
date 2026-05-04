# TLS certificates (local development)

**Do not commit** `*.pem` files. They are listed in `.gitignore`.

Generate a self-signed pair for HTTPS:

```bash
./scripts/gen_dev_certs.sh
```

Defaults expected by `cmd/web`:

- `certs/cert.pem`
- `certs/key.pem`

Override paths with `TLS_CERT_FILE` and `TLS_KEY_FILE` in `.env.local` if needed.

Browsers will warn about the self-signed certificate until you add an exception.
